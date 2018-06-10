package ons_manager

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"protobuf/ons_pb2"
	"sawtooth_sdk/processor"
	"sawtooth_sdk/logging"
	"ons/ons_state"
)

type Permission int32

const (
	PERMISSION_SU_ADDRESS Permission = iota+1
	PERMISSION_SU_MANAGER
	PERMISSION_MANAGER
	PERMISSION_NONE
)

var logger *logging.Logger = logging.Get()
var g_sudo_address string = "empty"
var g_ons_manager_address string = "empty"
var g_ons_manager *ons_pb2.ONSManager = &ons_pb2.ONSManager{}
var g_state_cached bool = false
var g_cached_ons_managers map[string]string = make(map[string]string)
var g_cached_ons_sumanagers map[string]bool = make(map[string]bool)

func SetSudoAddress(address string) bool {
	if g_sudo_address == "empty" {
		g_sudo_address = address
		return true
	}
	return false
}

func getONSManagerAddress() string {
	if g_ons_manager_address == "empty" {
		g_ons_manager_address = ons_state.GetNameSapce()+ons_state.Hexdigest("ons_manager")[:64]
	}
	return g_ons_manager_address;
}

func UnpackONSManager(ons_manager_byte_data []byte) (*ons_pb2.ONSManager, error) {
	ons_manager := &ons_pb2.ONSManager{}
	err := proto.Unmarshal(ons_manager_byte_data, ons_manager)
	logger.Debugf("UnpackONSManager:Unmarshal")

	if err != nil {
		return nil, &processor.InternalError{
			Msg: fmt.Sprint("Failed to unmarshal ONS Manager: %v", err)}
	}
	return ons_manager, nil
}

func LoadONSManager(context *processor.Context) (*ons_pb2.ONSManager, error) {
	//address로 state를 읽어 들인다 -> saveGS1Code에서 저장된 data이다.
	address := getONSManagerAddress()
	results, err := context.GetState([]string{address})
	if err != nil {
		logger.Debugf("LoadONSManager: address %v, error : %v", address, err)
		return nil, err
	}

	if len(string(results[address])) > 0 {
		ons_manager, err := UnpackONSManager(results[address])
		if err != nil {
			return nil, &processor.InvalidTransactionError{Msg: "Faied to UnpackGS1Code, address: " + address}
		}

		return ons_manager, nil
	}
	logger.Debugf("LoadONSManager: address %v doesn't exist", address)
	return nil, nil
}

func SaveONSManager(requestor string, ons_manager_data *ons_pb2.ONSManager, context *processor.Context) error {
	address := getONSManagerAddress()
	data, err := proto.Marshal(ons_manager_data)
	if err != nil {
		return &processor.InternalError{Msg: fmt.Sprint("Failed to serialize ONS Manager:", err)}
	}
	logger.Debugf("SaveONSManager data length : %v", len(data))

	addresses, err := context.SetState(map[string][]byte{
		address: data,
	})
	if err != nil {
		return err
	}

	if len(addresses) == 0 {
		return &processor.InternalError{Msg: "No addresses in set response"}
	}

	logger.Debugf("SaveONSManager address: " + address)
	return nil
}

func loadCachedONSManager(context *processor.Context) error {
	//context에서 manager address를 읽어와야 한다.
	//매번 읽을 수 없으니 caching으로..
	if g_state_cached == false {
		//state에서 ons manager를 읽어 들인다.
		//ons manager state address는 고정되어 있다.
		_manager, err := LoadONSManager(context)

		if err != nil {
			logger.Debugf("Failed to load ons manager, maybe not exist ")
			return err
		}

		if _manager == nil {
			logger.Debugf("ONS Manager doesn't exist in global state ")
			return nil
		}

		g_ons_manager = _manager

		//caching에서 찾기..
		for _, manager := range g_ons_manager.ManagerAddresses {
			g_cached_ons_managers[manager.GetGs1Code()] = manager.GetAddress()
		}

		for _, sumanager := range g_ons_manager.SuAddresses {
			g_cached_ons_sumanagers[sumanager.Address] = true
		}

		g_state_cached = true
	}
	return nil
}

func clearCachedONSManager() {
	//context에서 manager address를 읽어와야 한다.
	//매번 읽을 수 없으니 caching으로..
	if g_state_cached == true {
		g_ons_manager = nil
		g_cached_ons_managers = nil
		g_cached_ons_sumanagers = nil

		g_ons_manager = &ons_pb2.ONSManager{}
		g_cached_ons_managers = make(map[string]string)
		g_cached_ons_managers = make(map[string]string)
		g_state_cached = false
	}
}

func CheckPermission(gs1_code string, address string, context *processor.Context) (Permission, error) {
	if g_state_cached == false {
		err := loadCachedONSManager(context)
		if err != nil {
			return PERMISSION_NONE, err
		}
	}

	if address == g_sudo_address {
		logger.Debugf("You have su address auth")
		return PERMISSION_SU_ADDRESS, nil
	}

	if len(gs1_code) == 0 {
		return PERMISSION_NONE, nil
	}

	v, ok := g_cached_ons_sumanagers[address];
	if ok {
		if v == true {
			logger.Debugf("You have su manager auth")
			return PERMISSION_SU_MANAGER, nil
		}
	}

	vv, ok := g_cached_ons_managers[gs1_code];
	if ok {
		if vv == address {
			logger.Debugf("You have gs1 manager auth for %v", gs1_code)
			return PERMISSION_MANAGER, nil
		}
	}
	return PERMISSION_NONE, nil
}

func GetGS1CodeManagerAddress(gs1_code string, context *processor.Context) (string, bool, error) {
	//context에서 manager address를 읽어와야 한다.
	//매번 읽을 수 없으니 caching으로..
	if g_state_cached == false {
		err := loadCachedONSManager(context)
		if err != nil {
			return "", false, err
		}
	}
	v, ok := g_cached_ons_managers[gs1_code]
	return v, ok, nil
}

func AddGS1CodeManager(gs1_code string, address string, requestor string, context *processor.Context) error {
	//context에서 manager address를 읽어와야 한다.
	//매번 읽을 수 없으니 caching으로..
	if g_state_cached == false {
		err := loadCachedONSManager(context)
		if err != nil {
			return err
		}
	}

	//update or add address as gs1 code manager to cached ons managers
	if v, ok := g_cached_ons_managers[gs1_code]; ok {
		logger.Debugf("gs1 code manager already exist in the cache : %v", gs1_code)
		if v == address {
			logger.Debugf("gs1 code manager has the same address: %v", address)
			return SaveONSManager(requestor, g_ons_manager, context)
		}
	}

	g_cached_ons_managers[gs1_code] = address

	new_manager := &ons_pb2.ONSGS1CodeManager{
		Gs1Code: gs1_code,
		Address: address,
	}

	if g_ons_manager.ManagerAddresses == nil {
		g_ons_manager.ManagerAddresses = []*ons_pb2.ONSGS1CodeManager{new_manager}
	}else{
		for _, manager := range g_ons_manager.ManagerAddresses {
			if manager.Gs1Code == gs1_code {
				logger.Debugf("update gs1 code %s manager to %v from %v", gs1_code, address, manager.Address)
				manager.Address = address
				return SaveONSManager(requestor, g_ons_manager, context)
			}
		}
		g_ons_manager.ManagerAddresses = append(g_ons_manager.ManagerAddresses, new_manager)
	}

	return SaveONSManager(requestor, g_ons_manager, context)
}

func RemoveGS1CodeManager(gs1_code string, requestor string, context *processor.Context) error {
	//context에서 manager address를 읽어와야 한다.
	//매번 읽을 수 없으니 caching으로..
	if g_state_cached == false {
		err := loadCachedONSManager(context)
		if err != nil {
			return err
		}
	}

	if g_ons_manager.ManagerAddresses == nil {
		return &processor.InternalError{Msg: "ONS Managers doesn't exist"}
	}

	//update or add address as gs1 code manager to cached ons managers
	delete(g_cached_ons_managers, gs1_code)

	for idx, manager := range g_ons_manager.ManagerAddresses {
		if manager.Gs1Code == gs1_code {
			g_ons_manager.ManagerAddresses = append(g_ons_manager.ManagerAddresses[0:idx], g_ons_manager.ManagerAddresses[idx+1:]...)
		}
	}

	return SaveONSManager(requestor, g_ons_manager, context)
}

//just for test
func DeleteAllManager(context *processor.Context) error {
	address := getONSManagerAddress()
	clearCachedONSManager()
	_, err := context.DeleteState([]string{address})
	return err
}

func AddSuManager(su_address string, requestor string, context *processor.Context) error {
	//context에서 manager address를 읽어와야 한다.
	//매번 읽을 수 없으니 caching으로..
	if g_state_cached == false {
		err := loadCachedONSManager(context)
		if err != nil {
			return err
		}
	}

	//update or add address as gs1 code manager to cached ons managers
	if _, ok := g_cached_ons_sumanagers[su_address]; ok {
		logger.Debugf("su manager already exist in the cache : %v, so just call SaveONSManager with the same data", su_address)
		return SaveONSManager(requestor, g_ons_manager, context)
	}

	g_cached_ons_sumanagers[su_address] = true

	new_manager := &ons_pb2.ONSGS1CodeManager{
		Gs1Code: "",
		Address: su_address,
	}

	if g_ons_manager.SuAddresses == nil {
		g_ons_manager.SuAddresses =  []*ons_pb2.ONSGS1CodeManager{new_manager}
	}else{
		g_ons_manager.SuAddresses = append(g_ons_manager.SuAddresses, new_manager)
	}

	return SaveONSManager(requestor, g_ons_manager, context)
}

func RemoveSuManager(su_address string, requestor string, context *processor.Context) error {
	//context에서 manager address를 읽어와야 한다.
	//매번 읽을 수 없으니 caching으로..
	if g_state_cached == false {
		err := loadCachedONSManager(context)
		if err != nil {
			return err
		}
	}

	if g_ons_manager.SuAddresses == nil {
		return &processor.InternalError{Msg: "ONS Managers doesn't exist"}
	}

	//update or add address as gs1 code manager to cached ons managers
	delete(g_cached_ons_sumanagers, su_address)

	for idx, manager := range g_ons_manager.SuAddresses {
		if manager.Address == su_address {
			g_ons_manager.SuAddresses = append(g_ons_manager.SuAddresses[0:idx], g_ons_manager.SuAddresses[idx+1:]...)
		}
	}

	return SaveONSManager(requestor, g_ons_manager, context)
}

func OperateManager(op uint32, requestor string, context *processor.Context) error {
	//if op is 1, load cache...
	if requestor != g_sudo_address {
		return &processor.InvalidTransactionError{Msg: "LoadManager : Authentication failed"}
	}

	if op == 1 {
		logger.Debugf("OperateManager : load manager data for caching")
		clearCachedONSManager()
		err := loadCachedONSManager(context)
		if err != nil {
			return err
		}
	}

	return nil
}
