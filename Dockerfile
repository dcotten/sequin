ARG ELIXIR_VERSION=1.18.2
ARG OTP_VERSION=27.2.1
ARG DEBIAN_VERSION=buster-20240612-slim
ARG RELEASE_VERSION

ARG BUILDER_IMAGE="hexpm/elixir:${ELIXIR_VERSION}-erlang-${OTP_VERSION}-debian-${DEBIAN_VERSION}"
# Ensure Elixir installed on the runtime image for tooling purposes
ARG RUNNER_IMAGE="hexpm/elixir:${ELIXIR_VERSION}-erlang-${OTP_VERSION}-debian-${DEBIAN_VERSION}"

ARG SELF_HOSTED=0

# ---- Build Stage ----
FROM ${BUILDER_IMAGE} AS builder

# Pass the SELF_HOSTED arg as an environment variable
ARG SELF_HOSTED
ENV SELF_HOSTED=${SELF_HOSTED}

# Removed SENTRY_DSN arguments and environment variables
# ARG SENTRY_DSN
# ENV SENTRY_DSN=${SENTRY_DSN}

# Install build dependencies
RUN apt-get update -y && apt-get install -y build-essential git curl \
    && apt-get clean && rm -f /var/lib/apt/lists/*_*

# Install Node.js for build stage
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && apt-get install -y nodejs

# Prepare build directory
RUN mkdir /app
WORKDIR /app

# Install hex and rebar
RUN mix local.hex --force && \
    mix local.rebar --force

# Set environment variables for building the application
ENV MIX_ENV="prod"
ENV LANG=C.UTF-8
ENV ERL_FLAGS="+JPperf true"

# Install mix dependencies
COPY mix.exs mix.lock ./
RUN mix deps.get --only $MIX_ENV
RUN mkdir config

# Copy compile-time config files
COPY config/config.exs config/${MIX_ENV}.exs config/
RUN mix deps.compile

COPY priv priv
COPY lib lib

# Copy Git metadata (if needed elsewhere) and assets
COPY .git .git
COPY assets assets

# Install all npm packages in assets directory
WORKDIR /app/assets
RUN npm install

# Change back to build dir
WORKDIR /app

# Compile assets
RUN mix assets.deploy

# Pass through RELEASE_VERSION to the build environment
ARG RELEASE_VERSION
ENV RELEASE_VERSION=${RELEASE_VERSION}

# Compile the release
RUN mix compile

# Removed Sentry packaging command
# RUN mix sentry.package_source_code

# Changes to config/runtime.exs don't require recompiling the code
COPY config/runtime.exs config/

COPY rel rel
RUN mix release

# ---- App Stage ----
FROM ${RUNNER_IMAGE} AS app

# Install additional packages
RUN apt-get update -y && \
    apt-get install -y libstdc++6 openssl libncurses5 locales ca-certificates curl ssh jq telnet netcat htop \
    && apt-get clean && rm -f /var/lib/apt/lists/*_*

# Pass the SELF_HOSTED arg again in this stage
ARG SELF_HOSTED
ENV SELF_HOSTED=${SELF_HOSTED}

# Pass through RELEASE_VERSION to the runtime environment
ARG RELEASE_VERSION
ENV RELEASE_VERSION=${RELEASE_VERSION}

# Set the locale
RUN sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen && locale-gen
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Copy over the build artifact from the previous step and create a non-root user
RUN useradd --create-home app
WORKDIR /home/app
COPY --from=builder --chown=app /app/_build .

COPY .iex.exs .
RUN ln -s /home/app/prod/rel/sequin/bin/sequin /usr/local/bin/sequin
COPY scripts/start_commands.sh /scripts/start_commands.sh
RUN chmod +x /scripts/start_commands.sh

USER app

# Make port 4000 available to the world outside this container
EXPOSE 4000

# Run the start-up script which runs migrations and then the app
ENTRYPOINT ["/scripts/start_commands.sh"]