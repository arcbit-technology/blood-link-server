FROM golang:1.24.6 AS build_stage

WORKDIR /app

COPY go.sum go.mod ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 go build -o api/gopg-server ./main.go

FROM frolvlad/alpine-glibc

WORKDIR /app/

COPY --from=build_stage /app/api ./api/
COPY --from=build_stage /app/dao ./dao/
COPY --from=build_stage /app/config.docker.json ./api/config.json

EXPOSE 8080

CMD [ "./api/gopg-server"]
