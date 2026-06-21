FROM alpine:3.19

# Install essential tools
RUN apk add --no-cache \
    git \
    curl \
    bash \
    ca-certificates

# Create non-root user for security
RUN adduser -D -s /bin/bash sandbox

# Set working directory
WORKDIR /workspace

# Switch to non-root user
USER sandbox

# Default command
CMD ["/bin/bash"]