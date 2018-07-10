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


# 아래의 내용은 개인적인 메모입니다. (차후 삭제 예정)
* * *
*Transaction에서 사용하는 message에 map을 사용하면 block 생성시에 문제가 발생한다.
map을 사용하지 말고 repeated로 구현해야 한다.
protobuf의 messsage에서 map을 사용하는 것이 문제가 아니라,
global state에 저장하기 위해서 map을 marshaling해서 저장하면 marshaling 할 때마다
반환되는 byte data가 일정하지 않다(같은 data를 가지는 map이라도 변동이 생김)
이런 경우 global state에 저장할 때, 같은 data를 두번 마샬링해서 저장하는 경우 문제가 발생한다.*

*golang에서 channel을 사용할 때, read 하지 않는(<- [channel] code가 없는) channel에서 입력하는 경우([channel] <- value)
hang이 발생한다. -주의-*
