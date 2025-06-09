FROM golang:1.24.3-alpine3.21

# Install essential build dependencies
RUN apk add --no-cache \
    build-base \
    python3 \
    python3-dev \
    py3-pip \
    git \
    bash

# Set up working directory
WORKDIR /app/dysonprotocol

# Create and activate venv
RUN python3 -m venv /venv
ENV PATH="/venv/bin:$PATH"
ENV VIRTUAL_ENV="/venv"

# Install Python dependencies
COPY dev-requirements.txt ./
RUN . /venv/bin/activate && \
    pip install --no-cache-dir -U pip setuptools wheel && \
    pip install --no-cache-dir -r dev-requirements.txt

# Copy the project files
COPY . .

# Build DYSVM and install Dyson binary
RUN make dysvm
RUN make install

# Set dysond as the entry point to allow passing arguments
ENTRYPOINT ["sh"]
CMD [] 