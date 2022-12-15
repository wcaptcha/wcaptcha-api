# wcaptcha-api
Backend API of wCaptcha

## 私有化部署

wCaptcha 提供官方的服务，如果不想自己部署，可以直接到使用[官方的服务](https://wcaptcha.pingflash.com/)。

如果想要私有化部署，参考下面的操作步骤。

### 常规部署

安装前，请确保你已经安装了 go

直接从源代码构建
```shell
git clone https://github.com/wcaptcha/wcaptcha-api
cd wcaptcha-api
go build
```

也可在从 [Release 页面](https://github.com/wcaptcha/wcaptcha-api/releases)下载预先编译好的二进制文件

接下来，创建名为 `.env` 的配置文件，配置文件的内容可以参考 `.env.example` 

写好配置后，就可以直接运行了

```shell
./wcaptcha
```

接下来创建站点：
```shell
curl -X POST localhost:8090/site/create
```
返回结果为 JSON 格式的数据，其中包含 api_key 和 api_secret，这就是你的站点的密钥。

若要修改难度，可执行：
```shell
curl -X post localhost:8090/site/update --data "api_secret=YOUR_API_SECRET&hardness=HARDNESS
```
其中的 HARDNESS 是一个数字，站点创建后默认的 HARDNESS 是 `4194303`，即`2^22-1`，你可以填写任意数字。



### 部署到 Vercel

请注意，如果要部署到 vercel，则配置文件中不能使用 `file` 类型的存储。因为 vercel 是无服务模式，它的文件系统上内容是不会被保存的。

首先安装 vercel 的命令行：
```shell
npm i -g vercel
```

然后直接在代码目录下运行：

```shell
cp .env .env.production
./vercel-deploy.sh
```

之后按照提示操作即可。


