# ONS-Sawtooth

ONS-Sawtooth는 Hyperledger Sawtooth Blockchain으로 구현된 GS1 ONS(Object Name Service) project입니다.

GS1 ONS 기능을 구현하기 위해서 GS1 Code, Service record, Service type을 저장, 삭제, 수정 기능을 Sawtooth의 Transaction process로 구현했습니다.

## 시작하기

ONS-Sawtooth는 golang으로 구현되어 있으며 Hyperledger Sawtooth Blockchain상에서 Transaction process로 동작합니다.
따라서, 빌드를 하기 위해서는 golang과 Hyperledger Sawtooth Blockchaing을 설치해야 합니다.

참고로 이후의 모든 내용은 Ubuntun 16.0에서만 확인된 사항입니다.
MAC OS나 Windows OS에서는 아래 내용대로 동작하지 않을 수 있습니다.

## 필요사항

### golang 설치
설치 url : <https://golang.org/doc/install>을 참조하여 설치합니다.
golang은 GOROOT와 GOPATH 환경변수가 중요합니다. 주의깊게 읽고 설치하시기 바랍니다.
### Sawtooth 설치
설치 url : <https://sawtooth.hyperledger.org/docs/core/releases/latest/app_developers_guide/installing_sawtooth.html>를 참조
ubuntu 16.04인 경우에는 아래와 같이 설치할 수 있습니다.
```
$ sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 8AA7AF1F1091A5FD
$ sudo add-apt-repository 'deb http://repo.sawtooth.me/ubuntu/1.0/stable xenial universe'
$ sudo apt-get update
$ sudo apt-get install -y sawtooth
```
### Sawtooth 실행하기
ubuntu 16.04에서 실행하는 경우에는 [Using Sawtooth on Ubuntu 16.04](https://sawtooth.hyperledger.org/docs/core/releases/latest/app_developers_guide/ubuntu.html)를 참조하시면 됩니다.
docker나 AWS에서 실행하기를 원하는 경우에는 [Installing and Running Sawtooth](https://sawtooth.hyperledger.org/docs/core/releases/latest/app_developers_guide/installing_sawtooth.html)를 참조하시면 됩니다.

## ONS-Sawtooth source 다운받기
zmq4 package를 download하기전에 먼저 pkg-config와 libzmq를 설치한다.
```
$ sudo apt-get install pkg-config
$ sudo apt-get install libzmq5-dev
```
zmq, protobuf, ons-sawtooth-sdk, ons-sawtooth source를 download 받는다. 
```
$ go get -u github.com/jessevdk/go-flags
$ go get -u github.com/pebbe/zmq4
$ go get -u github.com/satori/go.uuid
$ go get -u github.com/golang/protobuf/proto
$ go get -u github.com/daludaluking/ons-sawtooth-sdk
$ go get -u github.com/daludaluking/ons-sawtooth
```

## ONS-Sawtooth Build 하기
ons-sawtooth source 경로로 이동한 후에 go build 명령어로 빌드한다.
```
$ cd $HOME/go/src/github.com/daludaluking/ons-sawtooth/src/ons
$ go build -o ./bin/ons
```
golang은 build를 하지 않고 바로 실행할 수 있다.
```
$ cd $HOME/go/src/github.com/daludaluking/ons-sawtooth/src/ons
$ go run main.go
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
