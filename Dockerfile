FROM golang

WORKDIR /

COPY . .

RUN go mod download
RUN go build -o .

CMD [ "./bot" ]