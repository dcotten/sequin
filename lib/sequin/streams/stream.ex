defmodule Sequin.Streams.Stream do
  @moduledoc false
  use Sequin.Schema

  import Ecto.Changeset
  import Ecto.Query

  alias Sequin.Accounts.Account
  alias Sequin.Streams.Stream

  @derive {Jason.Encoder, only: [:id, :slug, :account_id, :inserted_at, :updated_at]}
  typed_schema "streams" do
    field :slug, :string

    belongs_to :account, Account

    timestamps()
  end

  def changeset(%Stream{} = stream, attrs) do
    stream
    |> cast(attrs, [:slug])
    |> validate_required([:slug])
    |> validate_slug()
    |> unique_constraint([:account_id, :slug], error_key: :slug)
  end

  defp validate_slug(%Ecto.Changeset{valid?: false} = changeset), do: changeset

  defp validate_slug(%Ecto.Changeset{valid?: true, changes: %{slug: slug}} = changeset) do
    if String.match?(slug, ~r/^[a-zA-Z0-9_]+$/) do
      changeset
    else
      add_error(changeset, :slug, "must contain only alphanumeric characters or underscores")
    end
  end

  def where_id(query \\ base_query(), id) do
    from(s in query, where: s.id == ^id)
  end

  def where_account_id(query \\ __MODULE__, account_id) do
    from(s in query, where: s.account_id == ^account_id)
  end

  defp base_query(query \\ __MODULE__) do
    from(s in query, as: :stream)
  end
end
