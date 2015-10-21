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
docker run -d --env HURL="tcp://123.59.58.58:5002" --name="omega-logcollection" --net host -v /etc/omega/agent/omega-agent.conf:/etc/omega/agent/omega-agent.conf -v /var/run/docker.sock:/tmp/docker.sock -v /etc/localtime:/etc/localtime:ro registry.dataman.io/logspout  #使用自己指定的发段地址
```
如果想收集自己的日志，但是机器上没装agent需要一下格式命令
```
docker run -d --name="omega-logcollection" --net host --env CNAMES="/omega-slave,/omega-marathon,/omega-master,/omega-zookeeper" --env HOST_ID="111111" --env USER_ID="1" --env CLUSTER_ID="1"  -v /var/run/docker.sock:/tmp/docker.sock -v /etc/localtime:/etc/localtime:ro registry.dataman.io/logspout:omega.v0.2   #如果想收集的docker日志是通过mesos发布可以不加入CNAMES环境变量，如果收集日志的docker是手动启动的需要把docker的name通过CNAMES环境变量传进去
```
