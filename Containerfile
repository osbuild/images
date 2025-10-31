ARG BASE_CONTAINER_IMAGE=registry.fedoraproject.org/fedora:latest
FROM ${BASE_CONTAINER_IMAGE}

# The Fedora 41 container doesn't have python3 installed by default
RUN dnf install -y python3

WORKDIR /setup

COPY ./test/scripts ./test/scripts/
COPY Schutzfile .
RUN ./test/scripts/setup-osbuild-repo
RUN ./test/scripts/install-dependencies

COPY go.mod go.sum .
RUN go mod download

WORKDIR /app

# Mark the working directory as safe for git
RUN git config --global --add safe.directory /app
