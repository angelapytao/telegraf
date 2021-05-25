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
	FileDelCount int64
}

const MaxErrOffsetCount = 2000

var MapLogOffset sync.Map  //make(map[string]*LogOffset)

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

func (s *LogOffset) Set(offset,delCount int64) error {
    s.locker.Lock()
 	defer s.locker.Unlock()

    //文件删除或覆盖，如果旧的FileDelCount计数器小于当前计数器，说明是上一个文档的输出流（输出到kafka可能会阻塞或延迟）
    if s.Offset==0 && delCount<s.FileDelCount{
		return nil
	}
	offsetStr :=strconv.FormatInt(offset, 10)

	if offset==0&&s.file != nil {
		s.file.Close()
	}
	if offset==0||s.file == nil {
		file, err := os.OpenFile(s.FileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
		if err != nil {
			fmt.Println(err)
			return err
		}
		s.file = file
	}

	if offset>0 && s.Offset>=offset  {
		 return nil
	}

	_, err:= s.file.WriteAt([]byte(offsetStr), 0)
	if err != nil {
		return errors.New(s.file.Name() + err.Error())
	}
	s.Offset = offset
	s.FileDelCount = delCount
	return nil
}

func readFile(filename string) (int64,error){
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return 0, nil
	}
	fd, err :=  ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(filename,"read to fd fail", err)
		return -1,errors.New(filename+" "+ err.Error())
	}
	if string(fd)==""{
		return 0, nil
	}
	offset, err2 := strconv.ParseInt(string(fd), 10, 64)
	if err2 != nil {
		return -1, err2
	}
	return offset, nil
}

func (s *LogOffset) Close() error {
	s.locker.Lock()
	defer s.locker.Unlock()

	if s.file != nil {
		s.file.Close()
	}
	return nil
}

func SaveOffset(fileName string,offset,count int64) error{
	_logOffsetDto,ok:=MapLogOffset.Load(fileName)
	logOffsetDto := LogOffset{}
	if !ok {
		logOffsetDto.FileName = fileName + ".offset"
		MapLogOffset.Store(fileName, logOffsetDto)
	}else{
		logOffsetDto=_logOffsetDto.(LogOffset)
	}
	err := logOffsetDto.Set(offset,count)
    return err
}