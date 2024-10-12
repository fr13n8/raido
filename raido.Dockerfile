FROM golang:1.23

RUN apt-get update -y && apt-get upgrade -y
RUN apt-get install net-tools iproute2 iperf3 -y

WORKDIR /raido
COPY ./go.mod ./go.sum ./
RUN go mod download -x

COPY . /raido

RUN mkdir bin

RUN make raido OUTPUT=./bin/raido
RUN make agent OUTPUT=./bin/agent

ENV PATH="$PATH:/raido/bin"

RUN sysctl net.core.rmem_max=7500000
RUN sysctl net.core.wmem_max=7500000

WORKDIR /

CMD python3 -m http.server --bind :: 80