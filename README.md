# cgu
change-git-user 方便随心地切换git用户（使用Golang开发）

非常简单，几乎不需要记住命令：
```shell
$ cgu
cgu ls  查看当前目录git仓库所使用用户，及用户列表
cgu use 切换当前目录的git仓库用户
cgu add 添加git用户
cgu del 删除git用户
当前目录使用 name=sunhuawei email=sunhuawei@qq.com (作用于/Users/sunhuawei/coding/cgu)
+--------------+------------------------+
|    用户名     |          邮箱           |
+--------------+------------------------+
|   huaweiouba | sunhuawei250@gmail.com |
| * sunhuawei  | sunhuawei250@qq.com    |
+--------------+------------------------+
```

不需要记住复杂的命令，所有命令都是自动的
[![WHN4Bt.gif](https://z3.ax1x.com/2021/07/29/WHN4Bt.gif)](https://imgtu.com/i/WHN4Bt)

ps:
感谢 [toby](https://github.com/tob) 大佬的第一个star，没错，cgu里面炫酷的cli使用 [bubbletea](https://github.com/charmbracelet/bubbletea)