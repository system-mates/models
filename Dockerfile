ARG MATE_RUNTIME_IMAGE=ghcr.io/system-mates/runtime:latest
FROM ${MATE_RUNTIME_IMAGE}

# Add local models to the /mate/models folder
COPY system/models /mate/models
COPY example/models /mate/models

