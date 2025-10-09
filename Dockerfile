FROM alpine:latest

# Install required dependencies
RUN apk --no-cache add \
    ca-certificates \
    curl \
    tar \
    bash \
    ffmpeg \
    ripgrep

# Install uv (Python package manager)
RUN curl -LsSf https://astral.sh/uv/install.sh | sh
ENV PATH="/root/.local/bin:$PATH"

# Install yt-dlp
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp \
    -o /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp

WORKDIR /app

# Download and extract the latest release binary for Linux x64
RUN curl -L https://github.com/recreate-run/mix/releases/latest/download/mix-linux-x64.tar.gz -o mix.tar.gz && \
    tar -xzf mix.tar.gz && \
    chmod +x mix && \
    rm mix.tar.gz LICENSE README.md

# Expose port
EXPOSE 8080

# Create startup script that properly handles PORT environment variable
RUN printf '#!/bin/bash\nset -e\nPORT=${PORT:-8080}\necho "Starting Mix Agent on port $PORT..."\nexec ./mix --http-port "$PORT" --http-host 0.0.0.0 --dangerously-skip-permissions\n' > /app/start.sh && \
    chmod +x /app/start.sh

# Run the startup script
ENTRYPOINT ["/bin/bash", "/app/start.sh"]
