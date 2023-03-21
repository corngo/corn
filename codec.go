package infra

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	. "github.com/infrago/base"
	"github.com/infrago/util"
)

var (
	infraCodec = &codecModule{
		config: codecConfig{
			Text:  "01234AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz56789-_",
			Digit: "abcdefghijkmnpqrstuvwxyz123456789ACDEFGHJKLMNPQRSTUVWXYZ",
			Salt:  INFRA, Length: 7,

			Start:    time.Date(2022, 5, 1, 0, 0, 0, 0, time.Local),
			Timebits: 42, Nodebits: 7, Stepbits: 14,
			// 42bit=128年
		},
		codecs: make(map[string]Codec, 0),
	}
	errInvalidCodec     = errors.New("Invalid codec.")
	errInvalidCodecData = errors.New("Invalid codec data.")
)

const (
	JSON   = "json"
	XML    = "xml"
	GOB    = "gob"
	TOML   = "toml"
	DIGIT  = "digit"
	DIGITS = "digits"
	TEXT   = "text"
	TEXTS  = "text"
)

type (
	codecConfig struct {
		// Text Text 文本加密字母表
		Text string
		// Digit Digit 数字加密字母表
		Digit string
		// Salt 数字加密，加盐
		Salt string
		// Length 数字加密，最小长度
		Length int

		//雪花ID 开始时间
		Start time.Time
		//时间位
		Timebits uint
		//节点位
		Nodebits uint
		//序列位
		Stepbits uint
	}

	Codec struct {
		// Name 名称
		Name string
		// Text 说明
		Text string
		// Alias 别名
		Alias []string
		// Encode 编码方法
		Encode EncodeFunc
		// Decode 解码方法
		Decode DecodeFunc
	}
	EncodeFunc func(v Any) (Any, error)
	DecodeFunc func(d Any, v Any) (Any, error)

	// codecModule 是编解码模块
	// 主要用功能是 状态、多语言字串、MIME类型、正则表达式等等
	codecModule struct {
		mutex  sync.Mutex
		config codecConfig

		// codecs 编解码器集合
		codecs map[string]Codec

		fastid *util.FastID
	}
)

// Register
func (module *codecModule) Register(name string, value Any) {
	switch val := value.(type) {
	case Codec:
		module.Codec(name, val)
	}
}

// Configure
func (module *codecModule) Configure(global Map) {
	var config Map
	if vv, ok := global["codec"].(Map); ok {
		config = vv
	}

	//字串字符表
	if text, ok := config["text"].(string); ok {
		infraCodec.config.Text = text
	}

	//数字字母表
	if digit, ok := config["digit"].(string); ok {
		infraCodec.config.Digit = digit
	}
	if salt, ok := config["salt"].(string); ok {
		infraCodec.config.Salt = salt
	}
	if length, ok := config["length"].(int64); ok {
		infraCodec.config.Length = int(length)
	}
	if length, ok := config["length"].(int); ok {
		infraCodec.config.Length = int(length)
	}

	//雪花相关配置

	//开始时间
	if vv, ok := config["start"].(time.Time); ok {
		infraCodec.config.Start = vv
	}
	if vv, ok := config["start"].(int64); ok {
		infraCodec.config.Start = time.Unix(vv, 0)
	}
	//时间位
	if vv, ok := config["timebits"].(int); ok {
		infraCodec.config.Timebits = uint(vv)
	}
	if vv, ok := config["timebits"].(int64); ok {
		infraCodec.config.Timebits = uint(vv)
	}
	//节点位
	if vv, ok := config["nodebits"].(int); ok {
		infraCodec.config.Nodebits = uint(vv)
	}
	if vv, ok := config["nodebits"].(int64); ok {
		infraCodec.config.Nodebits = uint(vv)
	}
	//序列位
	if vv, ok := config["stepbits"].(int); ok {
		infraCodec.config.Stepbits = uint(vv)
	}
	if vv, ok := config["stepbits"].(int64); ok {
		infraCodec.config.Stepbits = uint(vv)
	}
}

func (this *codecModule) Initialize() {
	this.fastid = util.NewFastID(infraCodec.config.Timebits, infraCodec.config.Nodebits, infraCodec.config.Stepbits, infraCodec.config.Start.Unix())
}
func (module *codecModule) Connect() {
}
func (module *codecModule) Launch() {
}
func (module *codecModule) Terminate() {
}

// Codec 注册编解码器
func (this *codecModule) Codec(name string, config Codec) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}
	if config.Alias != nil {
		alias = append(alias, config.Alias...)
	}

	for _, key := range alias {
		if infra.override() {
			this.codecs[key] = config
		} else {
			if _, ok := this.codecs[key]; ok == false {
				this.codecs[key] = config
			}
		}
	}

}

func (this *codecModule) Codecs() map[string]Codec {
	codecs := map[string]Codec{}
	for k, v := range this.codecs {
		codecs[k] = v
	}

	return codecs
}

// Sequence 雪花ID
func (module *codecModule) Sequence() int64 {
	return infraCodec.fastid.NextID()
}

// Unique 雪花ID 转数字加密
func (module *codecModule) Generate(prefixs ...string) string {
	id := infraCodec.Sequence()
	ss, err := module.EncryptDIGIT(id)
	if err != nil {
		return fmt.Sprintf("%v", id)
	}
	if len(prefixs) > 0 {
		return fmt.Sprintf("%s%s", prefixs[0], ss)
	} else {
		return ss
	}
}

// Encode 原始的编码
func (module *codecModule) Encode(codec string, v Any) (Any, error) {
	codec = strings.ToLower(codec)
	if ccc, ok := infraCodec.codecs[codec]; ok {
		return ccc.Encode(v)
	}
	return nil, errInvalidCodec
}

// Decode 原始的解码
func (module *codecModule) Decode(codec string, d Any, v Any) (Any, error) {
	codec = strings.ToLower(codec)
	if ccc, ok := infraCodec.codecs[codec]; ok {
		return ccc.Decode(d, v)
	}
	return nil, errInvalidCodec
}

// Marshal 序列化
// 如 json, xml, gob 等
func (module *codecModule) Marshal(codec string, v Any) ([]byte, error) {
	dat, err := infraCodec.Encode(codec, v)
	if err != nil {
		return nil, err
	}
	if bts, ok := dat.([]byte); ok {
		return bts, nil
	}

	return nil, errInvalidCodecData
}

// Unmarshal 反序列化
// 如 json, xml, gob 等
func (module *codecModule) Unmarshal(codec string, d []byte, v Any) error {
	_, err := infraCodec.Decode(codec, d, v)
	return err
}

// Encrypt 数据加密
// 主要用类Var中的参数，数据
// 数据加密后，要返回明文可读的字串，方便传递
func (module *codecModule) Encrypt(codec string, v Any) (string, error) {
	dat, err := infraCodec.Encode(codec, v)
	if err != nil {
		return "", err
	}
	if bts, ok := dat.(string); ok {
		return bts, nil
	}
	if bts, ok := dat.([]byte); ok {
		return string(bts), nil
	} else {
		return fmt.Sprintf("%v", dat), nil
	}

	// return "", errInvalidCodecData
}

// Decrypt 数据解密
// 主要用类Var中的参数，数据
func (module *codecModule) Decrypt(codec string, v Any) (Any, error) {
	return infraCodec.Decode(codec, v, nil)
}

// MarshalJSON
func (module *codecModule) MarshalJSON(v Any) ([]byte, error) {
	return infraCodec.Marshal(JSON, v)
}

// UnmarshalJSON
func (module *codecModule) UnmarshalJSON(d []byte, v Any) error {
	return infraCodec.Unmarshal(JSON, d, v)
}

// MarshalXML
func (module *codecModule) MarshalXML(v Any) ([]byte, error) {
	return infraCodec.Marshal(XML, v)
}

// XMLUnmarshal
func (module *codecModule) UnmarshalXML(d []byte, v Any) error {
	return infraCodec.Unmarshal(XML, d, v)
}

// GOBMarshal
func (module *codecModule) MarshalGOB(v Any) ([]byte, error) {
	return infraCodec.Marshal(GOB, v)
}

// UnmarshalGOB
func (module *codecModule) UnmarshalGOB(d []byte, v Any) error {
	return infraCodec.Unmarshal(GOB, d, v)
}

// MarshalTOML
func (module *codecModule) MarshalTOML(v Any) ([]byte, error) {
	return infraCodec.Marshal(TOML, v)
}

// UnmarshalTOML
func (module *codecModule) UnmarshalTOML(d []byte, v Any) error {
	return infraCodec.Unmarshal(TOML, d, v)
}

// EncryptDIGIT
func (module *codecModule) EncryptDIGIT(n int64) (string, error) {
	return infraCodec.Encrypt(DIGIT, n)
}

// DigitsEncrypt alias for DigitsEncrypt
func (module *codecModule) EncryptDIGITS(ns []int64) (string, error) {
	return infraCodec.Encrypt(DIGITS, ns)
}

// DecryptDigit 解码数字
func (module *codecModule) DecryptDIGIT(s string) (int64, error) {
	val, err := infraCodec.Decrypt(DIGIT, s)
	if err != nil {
		return -1, err
	}

	if num, ok := val.(int); ok {
		return int64(num), nil
	}
	if num, ok := val.(int64); ok {
		return num, nil
	}

	return -1, errInvalidCodec
}

// DecryptDigits 解码数字数组
func (module *codecModule) DecryptDIGITS(s string) ([]int64, error) {
	val, err := infraCodec.Decrypt(DIGIT, s)
	if err != nil {
		return nil, err
	}

	if num, ok := val.(int); ok {
		return []int64{int64(num)}, nil
	}
	if num, ok := val.(int64); ok {
		return []int64{num}, nil
	}
	if num, ok := val.([]int64); ok {
		return num, nil
	}

	return nil, errInvalidCodec
}

// EncryptTEXT 加密文本
func (module *codecModule) EncryptTEXT(n string) (string, error) {
	return infraCodec.Encrypt(TEXT, n)
}

// TextsEncrypt 文本数组加密
func (module *codecModule) EncryptTEXTS(ns []string) (string, error) {
	return infraCodec.Encrypt(TEXTS, ns)
}

// DecryptTEXT 解码文本
func (module *codecModule) DecryptTEXT(s string) (string, error) {
	val, err := infraCodec.Decrypt(TEXT, s)
	if err != nil {
		return "", err
	}

	if sss, ok := val.(string); ok {
		return sss, nil
	} else if sss, ok := val.([]byte); ok {
		return string(sss), nil
	}

	return "", errInvalidCodec
}

// DecryptTEXTS 解码文本数组
func (module *codecModule) DecryptTEXTS(s string) ([]string, error) {
	val, err := infraCodec.Decrypt(TEXT, s)
	if err != nil {
		return nil, err
	}

	if num, ok := val.(string); ok {
		return []string{num}, nil
	}
	if num, ok := val.([]string); ok {
		return num, nil
	}

	return nil, errInvalidCodec
}

func TextAlphabet() string {
	return infraCodec.config.Text
}
func DigitAlphabet() string {
	return infraCodec.config.Digit
}
func DigitSalt() string {
	return infraCodec.config.Salt
}
func DigitLength() int {
	return infraCodec.config.Length
}

func Encode(name string, v Any) (Any, error) {
	return infraCodec.Encode(name, v)
}

func Decode(name string, data Any, obj Any) (Any, error) {
	return infraCodec.Decode(name, data, obj)
}

func Marshal(name string, obj Any) ([]byte, error) {
	return infraCodec.Marshal(name, obj)
}

func Unmarshal(name string, data []byte, obj Any) error {
	return infraCodec.Unmarshal(name, data, obj)
}

func Encrypt(name string, obj Any) (string, error) {
	return infraCodec.Encrypt(name, obj)
}

func Decrypt(name string, obj Any) (Any, error) {
	return infraCodec.Decrypt(name, obj)
}

//---------------------------------------

// MarshalJSON
func MarshalJSON(v Any) ([]byte, error) {
	return infraCodec.MarshalJSON(v)
}

// UnmarshalJSON
func UnmarshalJSON(d []byte, v Any) error {
	return infraCodec.UnmarshalJSON(d, v)
}

// MarshalXML
func MarshalXML(v Any) ([]byte, error) {
	return infraCodec.MarshalXML(v)
}

// XMLUnmarshal
func UnmarshalXML(d []byte, v Any) error {
	return infraCodec.UnmarshalXML(d, v)
}

// GOBMarshal
func MarshalGOB(v Any) ([]byte, error) {
	return infraCodec.MarshalGOB(v)
}

// UnmarshalGOB
func UnmarshalGOB(d []byte, v Any) error {
	return infraCodec.UnmarshalGOB(d, v)
}

// MarshalTOML
func MarshalTOML(v Any) ([]byte, error) {
	return infraCodec.MarshalTOML(v)
}

// UnmarshalTOML
func UnmarshalTOML(d []byte, v Any) error {
	return infraCodec.UnmarshalTOML(d, v)
}

// EncryptDIGIT
func EncryptDIGIT(n int64) (string, error) {
	return infraCodec.EncryptDIGIT(n)
}

// EncryptDIGITS
func EncryptDIGITS(ns []int64) (string, error) {
	return infraCodec.EncryptDIGITS(ns)
}

// DecryptDigit 解码数字
func DecryptDIGIT(s string) (int64, error) {
	return infraCodec.DecryptDIGIT(s)
}

// DecryptDigits 解码数字数组
func DecryptDIGITS(s string) ([]int64, error) {
	return infraCodec.DecryptDIGITS(s)
}

// EncryptTEXT 加密文本
func EncryptTEXT(n string) (string, error) {
	return infraCodec.EncryptTEXT(n)
}

// TextsEncrypt 文本数组加密
func EncryptTEXTS(ns []string) (string, error) {
	return infraCodec.EncryptTEXTS(ns)
}

// DecryptTEXT 解码文本
func DecryptTEXT(s string) (string, error) {
	return infraCodec.DecryptTEXT(s)
}

// DecryptTEXTS 解码文本数组
func DecryptTEXTS(s string) ([]string, error) {
	return infraCodec.DecryptTEXTS(s)
}

// Sequence 雪花ID
func Sequence() int64 {
	return infraCodec.Sequence()
}

// Unique 雪花ID 转数字加密
func Generate(prefixs ...string) string {
	return infraCodec.Generate(prefixs...)
}

//-----------------

func Codecs() map[string]Codec {
	return infraCodec.Codecs()
}
