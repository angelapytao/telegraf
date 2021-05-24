package log

import (
	"regexp"
	"strings"
)

type RegexpConfig struct {
	Name   string   `toml:"name"` //日志名称,如：app_info、web_info、app_warn、app_err
	File string `toml:"file"` //日志所在的文件名
	FileIsReg bool  `toml:"file_is_reg"` //文件名字段(File)是否为正则表达式
	Reg string `toml:"reg"` //匹配日志内容的正则表达式
}

type RegexpItem struct {
	Name  string `json:"name"`//日志名称,如：app_info、web_info、app_warn、app_err
	Reg string `json:"reg"`//匹配日志内容的正则表达式
}

func (reg *RegexpConfig) GetRegItems(filename string) *RegexpItem {
	item:=new(RegexpItem)
	if reg.FileIsReg{
		regxp:=regexp.MustCompile(reg.File)
		if regxp.MatchString(filename){
			item.Name=reg.Name
			item.Reg=reg.Reg
			return  item
		}
	}

	if strings.HasSuffix(filename,`/`+reg.File){
		item.Name=reg.Name
		item.Reg=reg.Reg
		return  item
	}
	return nil
}


//func getLogName(text string,regItems []RegexpItem )string{
//	logName:=""
//	for _,r:=range regItems{
//		reg:=regexp.MustCompile(r.Reg)
//		if reg.MatchString(text){
//			logName=r.Name
//			break
//		}
//	}
//	return  logName
//}

func isMatch(text string,regxp string )bool{
	reg:=regexp.MustCompile(regxp)
	if reg.MatchString(text){
		return true
	}
	return  false
}