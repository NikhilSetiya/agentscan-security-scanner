FROM alpine:3.18

# Install dependencies
RUN apk add --no-cache \
    git \
    bash \
    curl \
    make \
    gcc \
    musl-dev

# Install git-secrets
RUN git clone https://github.com/awslabs/git-secrets.git /tmp/git-secrets && \
    cd /tmp/git-secrets && \
    make install && \
    rm -rf /tmp/git-secrets

# Create a working directory
WORKDIR /repo

# Set git config to avoid warnings
RUN git config --global user.email "scanner@agentscan.com" && \
    git config --global user.name "AgentScan"

# Default command
CMD ["git", "secrets", "--help"]