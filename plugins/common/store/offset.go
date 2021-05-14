package store

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
)

type LogOffset struct {
	FileName string //文件名
	file     *os.File
	Offset   int64 //偏移量
	locker sync.Mutex
}

var MapLogOffset=make(map[string]*LogOffset)

func (s *LogOffset) Get() (int64, error) {
	s.locker.Lock()
	defer s.locker.Unlock()
	if s.file == nil {
		_, err := os.Stat(s.FileName)
		if os.IsNotExist(err) {
			return 0, nil
		}
	//	file, err := os.OpenFile(s.FileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
		file, err := os.OpenFile(s.FileName,  os.O_RDWR, 0666)
		if err != nil {
			fmt.Println(err)
			return -1, err
		}
		s.file = file
	}
	fd, err :=  ioutil.ReadFile(s.file.Name())
	if err != nil {
		fmt.Println("read to fd fail", err)
		return -1, err
	}
	if string(fd)==""{
		return 0, nil
	}
	offset, err2 := strconv.ParseInt(string(fd), 10, 64)
	if err2 == nil {
		s.Offset = offset
		return offset, nil
	}

	return -1, err2
}

func (s *LogOffset) Set(offset int64) error {
    s.locker.Lock()
 	defer s.locker.Unlock()

	offsetStr :=strconv.FormatInt(offset, 10)
	if s.file == nil {
		file, err := os.OpenFile(s.FileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
		if err != nil {
			fmt.Println(err)
			return err
		}
		s.file = file
	}
	////先判断原有偏移量是否最新的
	//fd, err := ioutil.ReadFile(s.file.Name())
	//if err != nil {
	//	fmt.Println(s.file.Name(),"read to fd fail", err)
	//	return  err
	//}
	//var offsetOld int64
	//if string(fd)!="" {
	//	offsetOld, err = strconv.ParseInt(string(fd), 10, 64)
	//	if err != nil {
	//		return errors.New(s.file.Name() + err .Error())
	//	}
	//}

	if s.Offset>=offset{
		return nil
	}
	_, err:= s.file.WriteAt([]byte(offsetStr), 0)
	if err != nil {
		return errors.New(s.file.Name() + err .Error())
	}
	s.Offset = offset
	return nil
}

func (s *LogOffset) Close() error {
	s.locker.Lock()
	defer s.locker.Unlock()

	if s.file != nil {
		s.file.Close()
	}
	return nil
}
