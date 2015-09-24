# logspout
数人云日志采集项目
##1.编译项目
```
git clone git@github.com:Dataman-Cloud/logspout.git
cd logspout && docker build -t dataman/logspout .
```
##2.1运行项目
```
docker run -d --name="omega-logcollection" --net host -v /etc/omega/agent/omega-agent.conf:/etc/omega/agent/omega-agent.conf -v /var/run/docker.sock:/tmp/docker.sock -v /etc/localtime:/etc/localtime:ro registry.dataman.io/logspout  #没有指定发送的地址使用默认
docker run -d --env hurl="tcp://123.59.58.58:5002" --name="omega-logcollection" --net host -v /etc/omega/agent/omega-agent.conf:/etc/omega/agent/omega-agent.conf -v /var/run/docker.sock:/tmp/docker.sock -v /etc/localtime:/etc/localtime:ro registry.dataman.io/logspout  #使用自己指定的发段地址
```
