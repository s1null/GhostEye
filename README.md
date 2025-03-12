# 🌐 GhostEye

<div align="center">
  
![Version](https://img.shields.io/badge/版本-1.0.0-blue)
![License](https://img.shields.io/badge/许可证-MIT-green)
![Platform](https://img.shields.io/badge/平台-Linux%20-lightgrey)

</div>

> GhostEye 是一个为渗透测试设计的专业Web Shell管理面板，它提供了一个安全、高效的远程终端访问界面，用于管理和控制目标系统。


## 🔍 背景与优势

传统的Shell管理工具存在多种问题，此工具就是为了简化红队人员在渗透测试过程中的繁琐操作而设计：

### ⚠️ 传统工具的局限性

- **🕒 网络延迟问题**：Tabby或直接SSH连接在高延迟网络环境下输入命令极其卡顿，影响操作效率
- **📶 连接稳定性差**：FinalShell虽然有独立输入框，但在网络波动情况下依然容易掉线，导致反弹shell中断
- **🔄 会话恢复繁琐**：虽然screen命令可以在掉线后恢复会话，但每次手动设置非常麻烦
- **🧩 反弹Shell流程复杂**：
  - 反弹shell命令种类繁多，需要借助HackTools等工具生成
  - 监听端口需要在SSH工具中手动设置，然后输入到命令生成工具，再复制执行
  - 部分环境下shell命令无法直接反弹，需要通过`echo ""|base64 -d |bash`间接执行
- **⌨️ 命令重复输入**：信息收集、权限提升、工具下载等常用命令需要反复手动输入，效率低下

---

## ✨ 特性

<table>
  <tr>
    <td>🚀 <b>一键监听</b></td>
    <td>自动设置监听端口，无需手动配置</td>
  </tr>
  <tr>
    <td>🔄 <b>命令同步</b></td>
    <td>端口自动同步到反弹命令中，省去手动修改步骤</td>
  </tr>
  <tr>
    <td>🔐 <b>编码转换</b></td>
    <td>一键生成base64反弹命令，满足各种环境需求</td>
  </tr>
  <tr>
    <td>⏱️ <b>持久会话</b></td>
    <td>后台保留shell会话，即使退出登录或重启浏览器也能立即恢复终端状态</td>
  </tr>
  <tr>
    <td>📶 <b>无惧网络波动</b></td>
    <td>专为不稳定网络环境设计，确保shell连接稳定可靠</td>
  </tr>
  <tr>
    <td>📚 <b>命令模板库</b></td>
    <td>保存和管理常用命令，如信息收集、提权、下载工具等，一键调用无需重复输入</td>
  </tr>
</table>

---

## 📥 Build

```bash
# 克隆仓库
git clone https://github.com/s1null/GhostEye.git

# 进入项目目录
cd GhostEye/src

# 构建项目
./build.sh
```

---

## 🛠️ 使用方法

GhostEye 支持多种命令行参数来自定义其行为：

```
./ghosteye [选项]

选项:
  -p string      服务器监听端口 (默认 "8080")
  -user string   指定管理员用户名
  -pass string   指定管理员密码
  -U int         自动生成指定数量的随机用户
  -w string      白名单IP地址，多个IP用逗号分隔
  --show-users   显示所有用户账号信息
```

### 📝 示例

<details>
<summary><b>点击展开使用示例</b></summary>

```bash
# 使用指定端口启动服务
./ghosteye -p 9090

# 指定管理员账号启动
./ghosteye -user admin -pass secure_password

# 启动并限制只允许特定IP访问
./ghosteye -w 192.168.1.100,10.0.0.5

# 生成5个随机用户并启动
./ghosteye -U 5

# 显示所有已创建的用户账号信息
./ghosteye --show-users
```
</details>


## 📋 使用示例

GhostEye允许您保存常用命令作为模板，极大提高工作效率：

### 🆕 添加新命令模板
- 在界面中点击"Add Command"按钮
- 填写命令名称、具体命令内容、命令描述
- 点击保存
![image](https://github.com/user-attachments/assets/bbddfdd8-97f4-4d20-bf6e-0fc3b588be8e)

### 🚀 快速调用
- 在终端会话中只需点击已保存的命令即可自动填充到输入区
- 可以先修改再执行，适应不同环境需求
- 点击base64会自动将输入框的命令转换为`echo ""|base64 -d |bash`
![image](https://github.com/user-attachments/assets/9774520d-5e3e-40a6-8e9e-0dee4e5574c4)

### 🔄 反弹Shell命令示例
![image](https://github.com/user-attachments/assets/980efb2e-bff7-40d7-96b0-7d7d944e378f)

---

## 🔑 登录系统

启动后，可以通过浏览器访问服务：

```
http://localhost:8080
```

如果没有指定管理员账号，系统会自动创建默认账号：
- 用户名: `admin`
- 密码: `admin`


---

## ⚖️ 免责声明

GhostEye 仅供合法的安全测试和教育目的使用。用户须遵守所有适用的法律法规，对因滥用本工具造成的任何后果自行承担全部责任。

---

<div align="center">
  
  <p>Made with ❤️ for Penetration Testers</p>
  <p>Copyright © 2025 s1null</p>
  
</div>
