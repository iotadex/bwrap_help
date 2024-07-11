result=`go version`
if [[ $result == "" ]] ; then
    echo -e "\e[31m !!! panic : golang is not installed"
    exit
fi

pkill bhelp
rm bhelp
rm -rf ./biota-swap
git clone https://github.com/TanglePay/biota-swap
cd biota-swap
go build -ldflags "-w -s"
cp bhelp ../bhelp_main
cd ..

if [ ! -f "./config/smpc_k" ];then
    echo -e "\e[31m !!! panic : Must cp the smpc_k file to the path of ./config/"
    exit
fi

./bhelp