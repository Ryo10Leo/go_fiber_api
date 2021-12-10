FROM golang:1.17
 
WORKDIR /app
# go mod init xxx でgo.modファイルを作成しておくこと
COPY go.mod .
COPY go.sum .
 
# go modからパッケージをダウンロード
RUN go mod download
 
# /app にすべてのコードをコピー
COPY . .
 
# Live Reloading
RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
 
# エントリポイント(air)
CMD ["air"]