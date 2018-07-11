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
zmq4 package를 download하기전에 먼저 pkg-config와 libzmq를 설치합니다.
```
$ sudo apt-get install pkg-config
$ sudo apt-get install libzmq5-dev
```
zmq, protobuf, ons-sawtooth-sdk, ons-sawtooth source를 download 받습니다. 
```
$ go get -u github.com/jessevdk/go-flags
$ go get -u github.com/pebbe/zmq4
$ go get -u github.com/satori/go.uuid
$ go get -u github.com/golang/protobuf/proto
$ go get -u github.com/daludaluking/ons-sawtooth-sdk
$ go get -u github.com/daludaluking/ons-sawtooth
```

## ONS-Sawtooth Build 하기
ONS-Sawtooth source 경로로 이동한 후에 go build 명령어로 빌드합니다.
```
$ cd $HOME/go/src/github.com/daludaluking/ons-sawtooth/src/ons
$ go build -o ./bin/ons
```
golang은 build를 하지 않고 바로 실행할 수 있습니다.
```
$ cd $HOME/go/src/github.com/daludaluking/ons-sawtooth/src/ons
$ go run main.go
```

## ONS-Sawtooth 실행하기
ONS-Sawtooth는 Hyperledger Sawtooth blockchain의 transaction process입니다.
Hyperledger Sawtooth blockchain의 validator가 실행되고 있을 때 ONS-sawtooth transaction process를 연결할 수 있습니다.

### Validator 실행하기
Validator를 실행하는 자세한 방법은 [](https://sawtooth.hyperledger.org/docs/core/nightly/master/app_developers_guide/ubuntu.html#step-4-generate-the-root-key-for-the-validator)를 참고하시기 바랍니다.
#### 1. Genesis block 만들기
```
$ sawset genesis
$ sudo -u sawtooth sawadm genesis config-genesis.batch
```
#### 2. Validator를 위한 root key 생성하기
```
$ sudo sawadm keygen
```
#### 3. Validator 실행하기
```
$ sudo -u sawtooth sawtooth-validator -vv
```
위와 같이 실행하면 tcp://localhost:4004로 component를 binding하게 됩니다.
이것은 REST API server와 transaction process가 component이며, tcp://localhost:4004로 component의 연결을 기다린다는 의미입니다.
외부에서의 연결을 지원하기 위해서는 아래와 같이 명령어로 실행하면 됩니다.
```
$ sudo -u sawtooth sawtooth-validator -vv --bind component:tcp://[ip address]:[port number]
```
#### 4. REST API server 실행 및 validator와 연결하기
REST API server는 validator에게 transaction(sawtooth에서는 batches라고 함)을 전달하고
분산 ledger의 state를 query하는 기능을 제공합니다.
```
$ sudo -u sawtooth sawtooth-rest-api -vv
```
만약, Validator를 실행할 때 --bind component option을 사용했다면 아래 명령어로 실행해야 합니다.
```
$ sudo -u sawtooth sawtooth-rest-api -vv --connect tcp://[ip address]:[port number]
```
REST API server를 외부에서 접속하도록 하기 위해서는 아래와 명령어로 실행해야 합니다.
```
$ sudo -u sawtooth sawtooth-rest-api -vv --bind [ip address]:[port number]
```
bind option으로 전달된 [[ip address]]:[[port_number]] 주소로 REST API를 호출할 수 있게 됩니다.
#### 5. Settings transaction processor 실행하기
Validator는 최초 실행이 되면 Settings transaction processor의 연결을 기다립니다.
Settings transaction processor가 연결되지 않으면 어떤 transaction도 수행되지 않기 때문에
반드시 Settings transaction processor를 실행해야 합니다.
```
$ sudo -u sawtooth settings-tp -vv
```
Validator를 실행할 때 --bind component option을 사용했다면 Settings transaction processor도 아래와 같은 명령어로 실행해야 합니다.
```
$ sudo -u sawtooth settings-tp -vv --connect tcp://[ip address]:[port number]
```
### ONS-Sawtooth 실행하기
ONS-Sawtooth를 빌드한 binary로 아래 명령어로 실행합니다.
```
$ ons -vv
```
만약, Validator를 실행할 때 --bind component option을 사용했다면 아래 명령어로 실행해야 합니다.
```
$ ons -vv --connect tcp://[ip address]:[port number]
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
