defmodule Sequin.DatabasesRuntime.Supervisor do
  @moduledoc """
  Supervisor for managing database-related runtime processes.
  """
  use Supervisor

  alias Sequin.Consumers.Backfill
  alias Sequin.Consumers.SinkConsumer
  alias Sequin.DatabasesRuntime.SlotProcessor
  alias Sequin.DatabasesRuntime.SlotProcessor.MessageHandler
  alias Sequin.DatabasesRuntime.SlotSupervisor
  alias Sequin.DatabasesRuntime.SlotSupervisorSupervisor
  alias Sequin.DatabasesRuntime.TableReaderServer
  alias Sequin.DatabasesRuntime.TableReaderServerSupervisor
  alias Sequin.DatabasesRuntime.WalEventSupervisor
  alias Sequin.DatabasesRuntime.WalPipelineServer
  alias Sequin.Replication.PostgresReplicationSlot
  alias Sequin.Repo

  require Logger

  defp table_reader_supervisor, do: {:via, :syn, {:replication, TableReaderServerSupervisor}}
  defp slot_supervisor, do: {:via, :syn, {:replication, SlotSupervisorSupervisor}}
  defp wal_event_supervisor, do: {:via, :syn, {:replication, WalEventSupervisor}}

  def start_link(opts) do
    name = Keyword.get(opts, :name, {:via, :syn, {:replication, __MODULE__}})
    Supervisor.start_link(__MODULE__, nil, name: name)
  end

  @impl Supervisor
  def init(_opts) do
    Supervisor.init(children(), strategy: :one_for_one)
  end

  def start_table_reader(supervisor \\ table_reader_supervisor(), %SinkConsumer{} = consumer, opts \\ []) do
    consumer = Repo.preload(consumer, [:active_backfill, :sequence])

    if is_nil(consumer.active_backfill) do
      Logger.warning("Consumer #{consumer.id} has no active backfill, skipping start")
    else
      default_opts = [
        backfill_id: consumer.active_backfill.id,
        table_oid: consumer.sequence.table_oid
      ]

      opts = Keyword.merge(default_opts, opts)

      Sequin.DynamicSupervisor.start_child(supervisor, {TableReaderServer, opts})
    end
  end

  def stop_table_reader(supervisor \\ table_reader_supervisor(), consumer)

  def stop_table_reader(_supervisor, %SinkConsumer{active_backfill: nil}) do
    :ok
  end

  def stop_table_reader(supervisor, %SinkConsumer{active_backfill: %Backfill{id: backfill_id}}) do
    Sequin.DynamicSupervisor.stop_child(supervisor, TableReaderServer.via_tuple(backfill_id))
    :ok
  end

  def stop_table_reader(supervisor, %SinkConsumer{} = consumer) do
    consumer
    |> Repo.preload(:active_backfill)
    |> stop_table_reader(supervisor)
  end

  def stop_table_reader(supervisor, backfill_id) when is_binary(backfill_id) do
    Sequin.DynamicSupervisor.stop_child(supervisor, TableReaderServer.via_tuple(backfill_id))
    :ok
  end

  def restart_table_reader(supervisor \\ table_reader_supervisor(), %SinkConsumer{} = consumer, opts \\ []) do
    stop_table_reader(supervisor, consumer)
    start_table_reader(supervisor, consumer, opts)
  end

  def start_replication(supervisor \\ slot_supervisor(), pg_replication_or_id, opts \\ [])

  def start_replication(_supervisor, %PostgresReplicationSlot{status: :disabled} = pg_replication, _opts) do
    Logger.info("PostgresReplicationSlot #{pg_replication.id} is disabled, skipping start")
    {:error, :disabled}
  end

  def start_replication(supervisor, %PostgresReplicationSlot{} = pg_replication, opts) do
    pg_replication = Repo.preload(pg_replication, [:postgres_database, sink_consumers: [:sequence]])
    opts = Keyword.put(opts, :pg_replication, pg_replication)
    test_pid = Keyword.get(opts, :test_pid)

    opts =
      Keyword.update(opts, :slot_message_store_opts, [test_pid: test_pid], fn opts ->
        Keyword.put(opts, :test_pid, test_pid)
      end)

    case Sequin.DynamicSupervisor.start_child(supervisor, {SlotSupervisor, opts}) do
      {:ok, pid} ->
        Logger.info("[DatabasesRuntime.Supervisor] Started SlotSupervisor", replication_id: pg_replication.id)
        SlotSupervisor.start_children(pg_replication, opts)
        {:ok, pid}

      {:ok, pid, _term} ->
        Logger.info("[DatabasesRuntime.Supervisor] Started SlotSupervisor", replication_id: pg_replication.id)
        SlotSupervisor.start_children(pg_replication, opts)
        {:ok, pid}

      {:error, {:already_started, pid}} ->
        SlotSupervisor.start_children(pg_replication, opts)
        {:ok, pid}

      {:error, error} ->
        Logger.error("[DatabasesRuntime.Supervisor] Failed to start SlotSupervisor", error: error)
        {:error, error}
    end
  end

  @spec refresh_message_handler_ctx(id :: String.t()) :: :ok | {:error, term()}
  def refresh_message_handler_ctx(id) do
    case Sequin.Replication.get_pg_replication(id) do
      {:ok, pg_replication} ->
        # Remove races by locking - better way?
        :global.trans(
          {__MODULE__, id},
          fn ->
            new_ctx = MessageHandler.context(pg_replication)

            case SlotProcessor.update_message_handler_ctx(id, new_ctx) do
              :ok -> :ok
              {:error, :not_running} -> :ok
            end
          end,
          [node() | Node.list()],
          # This is retries, not timeout
          20
        )

      error ->
        error
    end
  end

  def stop_replication(supervisor \\ slot_supervisor(), pg_replication_or_id)

  def stop_replication(supervisor, %PostgresReplicationSlot{id: id}) do
    stop_replication(supervisor, id)
  end

  def stop_replication(supervisor, id) do
    Logger.info("[DatabasesRuntime.Supervisor] Stopping replication #{id}")
    Sequin.DynamicSupervisor.stop_child(supervisor, SlotSupervisor.via_tuple(id))
  end

  def restart_replication(supervisor \\ slot_supervisor(), pg_replication_or_id) do
    stop_replication(supervisor, pg_replication_or_id)
    start_replication(supervisor, pg_replication_or_id)
  end

  def start_wal_pipeline_servers(supervisor \\ wal_event_supervisor(), pg_replication) do
    pg_replication = Repo.preload(pg_replication, :wal_pipelines)

    pg_replication.wal_pipelines
    |> Enum.filter(fn wp -> wp.status == :active end)
    |> Enum.group_by(fn wp -> {wp.replication_slot_id, wp.destination_oid, wp.destination_database_id} end)
    |> Enum.each(fn {{replication_slot_id, destination_oid, destination_database_id}, pipelines} ->
      start_wal_pipeline_server(supervisor, replication_slot_id, destination_oid, destination_database_id, pipelines)
    end)
  end

  def start_wal_pipeline_server(
        supervisor \\ wal_event_supervisor(),
        replication_slot_id,
        destination_oid,
        destination_database_id,
        pipelines
      ) do
    opts = [
      replication_slot_id: replication_slot_id,
      destination_oid: destination_oid,
      destination_database_id: destination_database_id,
      wal_pipeline_ids: Enum.map(pipelines, & &1.id)
    ]

    Sequin.DynamicSupervisor.start_child(
      supervisor,
      {WalPipelineServer, opts}
    )
  end

  def stop_wal_pipeline_servers(supervisor \\ wal_event_supervisor(), pg_replication) do
    pg_replication = Repo.preload(pg_replication, :wal_pipelines)

    pg_replication.wal_pipelines
    |> Enum.group_by(fn wp -> {wp.replication_slot_id, wp.destination_oid, wp.destination_database_id} end)
    |> Enum.each(fn {{replication_slot_id, destination_oid, destination_database_id}, _} ->
      stop_wal_pipeline_server(supervisor, replication_slot_id, destination_oid, destination_database_id)
    end)
  end

  def stop_wal_pipeline_server(
        supervisor \\ wal_event_supervisor(),
        replication_slot_id,
        destination_oid,
        destination_database_id
      ) do
    Logger.info("Stopping WalPipelineServer",
      replication_slot_id: replication_slot_id,
      destination_oid: destination_oid,
      destination_database_id: destination_database_id
    )

    Sequin.DynamicSupervisor.stop_child(
      supervisor,
      WalPipelineServer.via_tuple({replication_slot_id, destination_oid, destination_database_id})
    )
  end

  def restart_wal_pipeline_servers(supervisor \\ wal_event_supervisor(), pg_replication) do
    stop_wal_pipeline_servers(supervisor, pg_replication)
    start_wal_pipeline_servers(supervisor, pg_replication)
  end

  defp children do
    [
      Sequin.DatabasesRuntime.Starter,
      Sequin.DynamicSupervisor.child_spec(name: table_reader_supervisor()),
      Sequin.DynamicSupervisor.child_spec(name: slot_supervisor()),
      Sequin.DynamicSupervisor.child_spec(name: wal_event_supervisor())
    ]
  end
end
