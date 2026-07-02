FROM alpine:3.19

# Install essential tools
RUN apk add --no-cache \
    git \
    curl \
    bash \
    ca-certificates

# Create non-root user for security
RUN adduser -D -s /bin/bash sandbox

# Create directories for embedded files with proper ownership
RUN mkdir -p /workspace/.kiro/agents && \
    mkdir -p /workspace/.kiro/skills/github-cli && \
    mkdir -p /workspace/.kiro-krew/evals && \
    chown -R sandbox:sandbox /workspace/.kiro && \
    chown -R sandbox:sandbox /workspace/.kiro-krew

# Set working directory
WORKDIR /workspace

# Switch to non-root user
USER sandbox

# Default command
CMD ["/bin/bash"]