# ons-sawtooth
# Transaction에서 사용하는 message에 map을 사용하면 block 생성시에 문제가 발생한다.
# map을 사용하지 말고 repeated로 구현해야 한다.
# protobuf의 messsage에서 map을 사용하는 것이 문제가 아니라,
# global state에 저장하기 위해서 map을 marshaling해서 저장하면 marshaling 할 때마다
# 반환되는 byte data가 일정하지 않다(같은 data를 가지는 map이라도 변동이 생김)
# 이런 경우 global state에 저장할 때, 같은 data를 두번 마샬링해서 저장하는 경우 문제가 발생한다.
