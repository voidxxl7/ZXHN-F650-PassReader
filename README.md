# ZXHN-F650-PassReader
中兴光猫ZXHN F650超管密码获取工具

感谢：

​		[中兴光猫ZXHN-F650不拆机获取超级密码](https://www.52pojie.cn/thread-999381-1-1.html)  



​		[中兴光猫配置文件db_user_cfg.xml结构分析及解密](https://www.52pojie.cn/thread-1005978-1-1.html)  

  

链接中提到的获取方法，我测试时在同型号版本的光猫上失败，然后自己捣鼓了下发现在我这里需要用post方式提交参数才行(原帖的使用get就行，评论下测试失败的原因估计就是和我的一样)。  

~~因为一只蝙蝠的原因，便写了这个玩意，提交时get和post都使用了一遍(原帖作者也写了个工具，不过只是用get方式提交)，仅在以下设备型号版本下测试能获取：~~

* 设备类型：GPON天翼网关(4口单频)  
* 设备型号：ZXHN F650  
* 固件版本：V2.0.0P1T1  


**2023.12摸鱼更新，添加了新的解密方式，理论上支持更多的固件版本，但是没有任何的测试环境，所以行不行我也不知道...**

_____

##### 使用方法

仅适用于Windows和Linux

下载：<https://github.com/voidxxl7/ZXHN-F650-PassReader/releases>

直接运行然后输入普通用户的管理密码即可(默认的密码光猫背面有)  

_____


##### 命令行下使用
注意，需要有一定的基础，没有的话建议直接运行就行
```shell
$ ./ZXHN-F650-PassReader -help
Usage of ./ZXHN-F650-PassReader:
  -d    只下载db_user_cfg.xml到当前目录
  -f string
        从本地文件中读取并尝试解密
  -h string
        光猫登录ip，默认为192.168.1.1(不需要http://) (default "192.168.1.1")
  -help
        帮助
  -p string
        光猫的管理密码
  -u string
        光猫普通用户账号，默认为useradmin (default "useradmin")
```

登录光猫并输出超管密码

```shell
./F650-PassReader -p 密码 
```


只下载db_user_cfg.xml到当前目录
```shell
./F650-PassReader -p 密码 -d
```

读取本地的文件解密并保存到新的文件中，成功解密的文件将会保存在db_user_cfg_cbc.xml或者db_user_cfg_aescbc.xml
```shell
./F650-PassReader -f db_user_cfg.xml
```

_____

#### 编译
需要golang环境，支持交叉编译
```shell
# 生成windows可执行文件
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build .

# 生成linux可执行文件
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build .
```