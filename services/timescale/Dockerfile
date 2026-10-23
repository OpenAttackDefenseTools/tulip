FROM timescale/timescaledb:latest-pg15

RUN apk add build-base make clang15 llvm15 git
COPY sunflower /sunflower
RUN cd /sunflower && make USE_PGXS=1 with_llvm=no install && cd / && \
	git clone https://github.com/ossc-db/pg_hint_plan --branch PG15 && \
	cd pg_hint_plan && make USE_PGXS=1 with_llvm=no install
