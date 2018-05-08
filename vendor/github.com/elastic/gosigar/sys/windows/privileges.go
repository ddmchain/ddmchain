// +build windows

package windows

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

var (
	privNames     = make(map[string]int64)
	privNameMutex sync.Mutex
)

const (

	SeDebugPrivilege = "SeDebugPrivilege"
)

const (
	ERROR_NOT_ALL_ASSIGNED syscall.Errno = 1300
)

const (
	_SE_PRIVILEGE_ENABLED_BY_DEFAULT uint32 = 0x00000001
	_SE_PRIVILEGE_ENABLED            uint32 = 0x00000002
	_SE_PRIVILEGE_REMOVED            uint32 = 0x00000004
	_SE_PRIVILEGE_USED_FOR_ACCESS    uint32 = 0x80000000
)

type Privilege struct {
	LUID             int64  `json:"-"` 
	Name             string `json:"-"`
	EnabledByDefault bool   `json:"enabled_by_default,omitempty"`
	Enabled          bool   `json:"enabled"`
	Removed          bool   `json:"removed,omitempty"`
	Used             bool   `json:"used,omitempty"`
}

func (p Privilege) String() string {
	var buf bytes.Buffer
	buf.WriteString(p.Name)
	buf.WriteString("=(")

	opts := make([]string, 0, 4)
	if p.EnabledByDefault {
		opts = append(opts, "Default")
	}
	if p.Enabled {
		opts = append(opts, "Enabled")
	}
	if !p.EnabledByDefault && !p.Enabled {
		opts = append(opts, "Disabled")
	}
	if p.Removed {
		opts = append(opts, "Removed")
	}
	if p.Used {
		opts = append(opts, "Used")
	}

	buf.WriteString(strings.Join(opts, ", "))
	buf.WriteString(")")

	return buf.String()
}

type User struct {
	SID     string
	Account string
	Domain  string
	Type    uint32
}

func (u User) String() string {
	return fmt.Sprintf(`User:%v\%v, SID:%v, Type:%v`, u.Domain, u.Account, u.SID, u.Type)
}

type DebugInfo struct {
	OSVersion    Version              
	Arch         string               
	NumCPU       int                  
	User         User                 
	ProcessPrivs map[string]Privilege 
}

func (d DebugInfo) String() string {
	bytes, _ := json.Marshal(d)
	return string(bytes)
}

func LookupPrivilegeName(systemName string, luid int64) (string, error) {
	buf := make([]uint16, 256)
	bufSize := uint32(len(buf))
	err := _LookupPrivilegeName(systemName, &luid, &buf[0], &bufSize)
	if err != nil {
		return "", errors.Wrapf(err, "LookupPrivilegeName failed for luid=%v", luid)
	}

	return syscall.UTF16ToString(buf), nil
}

func mapPrivileges(names []string) ([]int64, error) {
	var privileges []int64
	privNameMutex.Lock()
	defer privNameMutex.Unlock()
	for _, name := range names {
		p, ok := privNames[name]
		if !ok {
			err := _LookupPrivilegeValue("", name, &p)
			if err != nil {
				return nil, errors.Wrapf(err, "LookupPrivilegeValue failed on '%v'", name)
			}
			privNames[name] = p
		}
		privileges = append(privileges, p)
	}
	return privileges, nil
}

func EnableTokenPrivileges(token syscall.Token, privileges ...string) error {
	privValues, err := mapPrivileges(privileges)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint32(len(privValues)))
	for _, p := range privValues {
		binary.Write(&b, binary.LittleEndian, p)
		binary.Write(&b, binary.LittleEndian, uint32(_SE_PRIVILEGE_ENABLED))
	}

	success, err := _AdjustTokenPrivileges(token, false, &b.Bytes()[0], uint32(b.Len()), nil, nil)
	if !success {
		return err
	}
	if err == ERROR_NOT_ALL_ASSIGNED {
		return errors.Wrap(err, "error not all privileges were assigned")
	}

	return nil
}

func GetTokenPrivileges(token syscall.Token) (map[string]Privilege, error) {

	var size uint32
	syscall.GetTokenInformation(token, syscall.TokenPrivileges, nil, 0, &size)

	b := bytes.NewBuffer(make([]byte, size))
	err := syscall.GetTokenInformation(token, syscall.TokenPrivileges, &b.Bytes()[0], uint32(b.Len()), &size)
	if err != nil {
		return nil, errors.Wrap(err, "GetTokenInformation failed")
	}

	var privilegeCount uint32
	err = binary.Read(b, binary.LittleEndian, &privilegeCount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read PrivilegeCount")
	}

	rtn := make(map[string]Privilege, privilegeCount)
	for i := 0; i < int(privilegeCount); i++ {
		var luid int64
		err = binary.Read(b, binary.LittleEndian, &luid)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read LUID value")
		}

		var attributes uint32
		err = binary.Read(b, binary.LittleEndian, &attributes)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read attributes")
		}

		name, err := LookupPrivilegeName("", luid)
		if err != nil {
			return nil, errors.Wrapf(err, "LookupPrivilegeName failed for LUID=%v", luid)
		}

		rtn[name] = Privilege{
			LUID:             luid,
			Name:             name,
			EnabledByDefault: (attributes & _SE_PRIVILEGE_ENABLED_BY_DEFAULT) > 0,
			Enabled:          (attributes & _SE_PRIVILEGE_ENABLED) > 0,
			Removed:          (attributes & _SE_PRIVILEGE_REMOVED) > 0,
			Used:             (attributes & _SE_PRIVILEGE_USED_FOR_ACCESS) > 0,
		}
	}

	return rtn, nil
}

func GetTokenUser(token syscall.Token) (User, error) {
	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return User{}, errors.Wrap(err, "GetTokenUser failed")
	}

	var user User
	user.SID, err = tokenUser.User.Sid.String()
	if err != nil {
		return user, errors.Wrap(err, "ConvertSidToStringSid failed")
	}

	user.Account, user.Domain, user.Type, err = tokenUser.User.Sid.LookupAccount("")
	if err != nil {
		return user, errors.Wrap(err, "LookupAccountSid failed")
	}

	return user, nil
}

func GetDebugInfo() (*DebugInfo, error) {
	h, err := windows.GetCurrentProcess()
	if err != nil {
		return nil, err
	}

	var token syscall.Token
	err = syscall.OpenProcessToken(syscall.Handle(h), syscall.TOKEN_QUERY, &token)
	if err != nil {
		return nil, err
	}

	privs, err := GetTokenPrivileges(token)
	if err != nil {
		return nil, err
	}

	user, err := GetTokenUser(token)
	if err != nil {
		return nil, err
	}

	return &DebugInfo{
		User:         user,
		ProcessPrivs: privs,
		OSVersion:    GetWindowsVersion(),
		Arch:         runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
	}, nil
}
