FROM buildpack-deps:jessie-scm

# More stuff for bleve
# http://www.blevesearch.com/docs/Building/
RUN apt-get update && apt-get install -y --no-install-recommends \
        g++ \
        gcc \
        libc6-dev \
        make \
        pkg-config \
        libleveldb-dev \
        libstemmer-dev \
        libicu-dev \
        build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /usr/local/bin
WORKDIR /usr/local/bin
COPY biblescholar-linux-amd64 /usr/local/bin/

# Just copy data in
COPY verses.bleve /data/verses.bleve

EXPOSE 80

# /data/verses.bleve should be mounted as a volume
CMD ["/usr/local/bin/biblescholar-linux-amd64", "server", "-p", "80", "-i", "/data/verses.bleve", "--validate-alexa"]
