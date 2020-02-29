# ZXHN-F650-PassReader
中兴光猫ZXHN F650超管密码获取工具

感谢：

​		[中兴光猫ZXHN-F650不拆机获取超级密码](https://www.52pojie.cn/thread-999381-1-1.html)  



​		[中兴光猫配置文件db_user_cfg.xml结构分析及解密](https://www.52pojie.cn/thread-1005978-1-1.html)  

  

链接中提到的获取方法，我测试时在同型号版本的光猫上失败，然后自己捣鼓了下发现在我这里需要用post方式提交参数才行(原帖的使用get就行，评论下测试失败的原因估计就是和我的一样)。  

因为一只蝙蝠的原因，便写了这个玩意，提交时get和post都使用了一遍(原帖作者也写了个工具，不过只是用get方式提交)，仅在以下设备型号版本下测试能获取：   

* 设备类型：GPON天翼网关(4口单频)  
* 设备型号：ZXHN F650  
* 固件版本：V2.0.0P1T1  

  

##### 使用方法

仅适用于Windows，其他系统请自行安装golang环境

下载：<https://github.com/voidxxl7/ZXHN-F650-PassReader/releases>

直接运行然后输入普通用户的管理密码即可(默认的密码光猫背面有)  

  

在终端下使用命令

```shell
F650-PassReader -p 密码 -s
```

可以把解密后的db_user_cfg.xml配置文件保存在当前目录下。

