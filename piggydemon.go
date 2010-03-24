package main

import "fmt"
import "net"
import "bytes"
import "strings"
import "exec"
import "os"
import "io"
import "io/ioutil"
import "bufio"
import "regexp"
//import "container/vector"

const (
	INPORT    int   = 9999 // port to bind this program to
	RCPORT    int   = 9999 // port, renderclients listen at
	INBUFSIZE int   = 512
	TIMEOUT   int64 = 9000000000 // 9 sec timeout
	//	RENDERFARMPATH string = "/storage/renderfarm/"
	RENDERFARMPATH string = "./storage/renderfarm/"
	CONFIGPATH     string = RENDERFARMPATH + "configs/"
	ETHERWAKE string = "/usr/local/sbin/etherwake"
)


func main() {

	fmt.Printf("Hello, 世界\n")


	// socket einrichten und auf verbindung warten
	var listenAddr *net.TCPAddr = new(net.TCPAddr)
	listenAddr.Port = INPORT
	listenSock, err := net.ListenTCP("tcp4", listenAddr)
	if err != nil {
		fmt.Printf(err.String())
	}

	for {
		//establisch connection
		conn, _ := listenSock.AcceptTCP()
		go handleConnection(conn) // add 'go' to allow multiple connections
	}
}

func getRidOfDummies(inStrings []string) (outStrings []string) {
	outStrings = make([]string, len(inStrings))
	i := 0
	for _, k := range inStrings {
		if string(k) != "" {
			outStrings[i] = k
			i++
		}
	}
	outStrings = outStrings[0:i]
	return
}

func handleConnection(listenConn *net.TCPConn) {
	listenConn.SetReadBuffer(INBUFSIZE)
	listenConn.SetTimeout(TIMEOUT)

	bufioReader := bufio.NewReader(listenConn)
	bufioWriter := bufio.NewWriter(listenConn)

	addr := listenConn.RemoteAddr()
	fmt.Printf("\nnetwork: %s\n", addr.Network())
	fmt.Printf("\naddr: %s\n", addr.String())

	for {
		fmt.Printf("\nSCHLEIFE\n")
		request, e := bufioReader.ReadBytes('!')
		if e != nil {
			break
		}
		requestLines := strings.Split(string(request), ";", 0)
		length := len(requestLines)
		if length > 0 {
			length--
		}
		requestLines = requestLines[0:length]
		for i, k := range requestLines {
			fmt.Printf("requestLines[%d]: \"%s\"\n", i, k)
		}
		infos := parseCommand(len(requestLines), requestLines)
		fmt.Printf("\nINFOS:\n")
		for i, k := range infos {
			fmt.Printf("\ninfos[%d]\"%s\"\n", i, k)
		}
		answer := strings.Join(infos, ";")
		answer = requestLines[0] + ";" + answer + ";!"
		fmt.Printf("\nSending: \"%s\"\n", answer)
		bufioWriter.WriteString(answer)
		bufioWriter.Flush()
		fmt.Printf("\nSENT\n")
	}
	listenConn.Close()
}

func parseCommand(argc int, argv []string) (result []string) {
	if len(argv) < 1 {
		return []string{"#ungueltige Anfrage erhalten"}
	}
	funktion := strings.ToLower(argv[0])
	fmt.Printf("\n\nFunktion: \"%s\" \"%s\" ", argv[0], funktion)
	//fmt.Printf("\nlen funktion %d %d", len(funktion),len(strings.TrimSpace("getclients")) )
	switch funktion {
	case "getclients":
		fmt.Printf("\nCALL GETCLIENTS\n")
		// DUMMY:
		result = callGetClients(argc, argv)
	case "getimages":
		fmt.Printf("CALL GETIMAGES\n")
		// DUMMY:
		//result = []string{"fixed_overlay"}
		result = callGetImages(argc, argv)
	case "get":
		fmt.Printf("CALL GET\n")
		// to be redirected to calling client
		result = callGet(argc, argv)
	case "set":
		fmt.Printf("CALL SET\n")
		result = callSet(argc, argv)
	case "shutdown":
		fmt.Printf("CALL SHUTDOWN\n")
		result = callShutdownRebootState(argc, argv)
	case "reboot":
		fmt.Printf("CALL REBOOT\n")
		result = callShutdownRebootState(argc, argv)
	case "wol":
		fmt.Printf("CALL WOL\n")
		result = callWOL(argc, argv)
	/*case "vnc":
	fmt.Printf("CALL VNC\n")*/
	/*case "copy":
	fmt.Printf("CALL COPY\n")*/
	case "querystate":
		fmt.Printf("CALL QUERYSTATE\n")
	default:
		fmt.Printf("KEINE FUNKTION GEFUNDEN")
		result = []string{"#keine funktion gefunden"}
	}
	return
	//return "unknown feature. use get,set,shutdown,reboot,vnc,copy,querystate"
}
// get;render-23;192.168.1.123;255.255.255.0;192.168.1.254;192.168.0.1211;base1;00:11:22:33:44:55;001122334455;!
func callWOL(argc int, argv []string) (result []string) {
	rechnerInfo := getInfoByRechnername(argv[1])
	p,e := exec.Run(ETHERWAKE, []string{ETHERWAKE, rechnerInfo[6]}, nil, "/", exec.DevNull, exec.DevNull, exec.DevNull)
	if e != nil {
		return []string{"#Error calling etherwake"}
	}
	p.Wait(0)
	return []string{"OK"}
}

func callShutdownRebootState(argc int, argv []string) (result []string) {
	result = []string{"#ip nicht angegeben"}
	if (len(argv) > 1) && (argv[1] != "") {
		rcAddr, e := net.ResolveTCPAddr(argv[1])
		if e != nil {
			return []string{"#ungueltige ip"}
		}
		lAddr, _ := net.ResolveTCPAddr("")
		rcConn, e := net.DialTCP("tcp4", lAddr, rcAddr)
		if e != nil {
			return []string{"#verbindung nicht abgebaut"}
		}
		_, e = rcConn.Write([]byte(argv[0] + ";!"))
		if e != nil {
			return []string{"#fehler beim senden an den renderclient"}
		}

		bufioReader := bufio.NewReader(rcConn)
		rrAnswer, e := bufioReader.ReadBytes('!')
		if e != nil {
			return []string{"#fehler beim empfangen der antwort"}
		}

		rrAnswerLines := strings.Split(string(rrAnswer), ";", 0)
		return rrAnswerLines

		rcConn.Close()
	}
	return
}

func callSet2(argc int, argv []string) (result []string) {
	result = []string{"#rechnername nicht angegeben"}
	//neuen Rechnernamen aus dem request entfernen
	newArgv := make([]string, len(argv)-1)
	i := 0
	for j, k := range argv {
		if j != 2 {
			newArgv[i] = k
			i++
		}
	}
	// query mit altem rechnernamen
	rechnerInfos := callGet(argc-1, newArgv)
	if len(rechnerInfos) != 8 {
		return
	}
	macPureString := rechnerInfos[7]
	ipAlt := rechnerInfos[1]
	ipNeu := argv[3]
	result[0] = setIp(ipAlt, ipNeu, macPureString)
	return
}

func callSet(argc int, argv []string) (result []string) {
	result = []string{"#zu wenige argumente"}
	if len(argv) < 8 {
		return
	}
	//neuen Rechnernamen aus dem request entfernen
	newArgv := make([]string, len(argv)-1)
	i := 0
	for j, k := range argv {
		if j != 2 {
			newArgv[i] = k
			i++
		}
	}
	// query mit altem rechnernamen
	rechnerInfos := callGet(argc-1, newArgv)
	if len(rechnerInfos) != 8 {
		return
	}
	//ipAlt := rechnerInfos[1]
	//result[0] = setIp(ipAlt,ipNeu,macPureString)
	//result := []string{ nameString,
	//		ipString,
	//		subnetString,
	//		gatewayString,
	//		dnsString,
	//		imageString,
	//		macString,
	//		macPureString }
	nameAlt := argv[1]
	macPureAlt := rechnerInfos[7]
	nameNeu := argv[2]
	ipNeu := argv[3]
	subnetNeu := argv[4]
	gatewayNeu := argv[5]
	dnsNeu := argv[6]
	imageNeu := argv[7]
	macNeu := macPureAlt
	macPureNeu := strings.Join(strings.Split(macNeu, ":", 0), "")
	fmt.Printf("macNeu: %s\n", macNeu)
	fmt.Printf("macPureNeu: %s\n", macPureNeu)

	//  if macPureNeu != macPureAlt -> mv directory
	if macPureNeu != macPureAlt {
		runSystemCommand([]string{"mv", macPureAlt, macPureNeu}, RENDERFARMPATH+"configs/")
		runSystemCommand([]string{"mv", macPureAlt, macPureNeu}, RENDERFARMPATH+"overlays/")
	}
	// write overlays/:::::/network/interfaces
	filePath := RENDERFARMPATH + "overlays/" + macPureNeu + "/etc/network/interfaces"
	fd, e := os.Open(filePath, os.O_WRONLY|os.O_CREAT, 0)
	if e != nil {
		result = []string{"#could not open file " + filePath}
		return
	}
	_, e = fd.WriteString("auto lo\n" +
		"iface lo inet network\n" +
		"\n" +
		"auto eth0\n" +
		"iface eth0 inet static\n" +
		"\taddress " + ipNeu + "\n" +
		"\tnetmask " + subnetNeu + "\n" +
		"\tgateway " + gatewayNeu + "\n" +
		"\tdns-nameserver " + dnsNeu + "\n")
	if e != nil {
		result = []string{"#could not write file " + filePath}
		return
	}
	fd.Close()
	// write overlays/:::::/hostname
	filePath = RENDERFARMPATH + "overlays/" + macPureNeu + "/hostname"
	fd, e = os.Open(filePath, os.O_WRONLY|os.O_CREAT, 0)
	if e != nil {
		result = []string{"#could not open file " + filePath}
		return
	}
	_, e = fd.WriteString(nameNeu)
	if e != nil {
		result = []string{"#could not write file " + filePath}
		return
	}
	fd.Close()
	// write configs/:::::/config
	filePath = RENDERFARMPATH + "configs/" + macPureNeu + "/config"
	fd, e = os.Open(filePath, os.O_WRONLY|os.O_CREAT, 0)
	if e != nil {
		result = []string{"#could not open file " + filePath}
		return
	}
	_, e = fd.WriteString(imageNeu + "\nnone")
	if e != nil {
		result = []string{"#could not write file " + filePath}
		return
	}
	fd.Close()
	// write configs/:::::/config
	filePath = RENDERFARMPATH + "configs/" + macPureNeu + "/network"
	fd, e = os.Open(filePath, os.O_WRONLY|os.O_CREAT, 0)
	if e != nil {
		result = []string{"#could not open file " + filePath}
		return
	}
	_, e = fd.WriteString(ipNeu + "\n" + nameNeu)
	if e != nil {
		result = []string{"#could not write file " + filePath}
		return
	}
	fd.Close()

	result = []string{"set", nameAlt, nameNeu, ipNeu, subnetNeu, gatewayNeu, dnsNeu, imageNeu, macNeu}
	return
}

func setIp(ipOld, ipNew, mac string) (errMsg string) {
	// alte ip als regexp
	//exp := regexp.Compile("s/"+ipOld+"//g")
	exp, _ := regexp.Compile("address " + ipOld)
	// erste datei aendern
	filePath := RENDERFARMPATH + "overlays/" + mac + "/network/interfaces"
	inhalt, e := ioutil.ReadFile(filePath)
	if e != nil {
		errMsg = "#couldn't open file " + filePath
	}
	fmt.Printf("INHALT vorher:\n%s\n", string(inhalt))
	ipNewBuf := bytes.NewBufferString("address " + ipNew)
	inhalt = exp.ReplaceAll(inhalt, ipNewBuf.Bytes())
	fmt.Printf("INHALT nachher:\n%s\n", string(inhalt))
	// write auf erste datei
	e = ioutil.WriteFile(filePath, inhalt, 0)
	if e != nil {
		errMsg = "#could not write to file " + filePath + "\n"
		return
	}
	exp, _ = regexp.Compile(ipOld)
	// zweite datei aendern
	filePath = RENDERFARMPATH + "configs/" + mac + "/network"
	inhalt, e = ioutil.ReadFile(filePath)
	if e != nil {
		errMsg = "#couldn't open file " + filePath
	}
	fmt.Printf("INHALT vorher:\n%s\n", string(inhalt))
	ipNewBuf = bytes.NewBufferString(ipNew)
	inhalt = exp.ReplaceAll(inhalt, ipNewBuf.Bytes())
	fmt.Printf("INHALT nachher:\n%s\n", string(inhalt))
	// write auf erste datei
	e = ioutil.WriteFile(filePath, inhalt, 0)
	if e != nil {
		errMsg = "#could not write to file " + filePath + "\n"
	}
	return
}


func callGet(argc int, argv []string) []string {
	if (len(argv) > 1) && (argv[1] != "") {
		fmt.Printf("RECHNERNAME: \"%s\"\n", argv[1])
		return getInfoByRechnername(argv[1])
	}
	return []string{"#rechnername nicht angegeben"}
}

func callGetImages(argc int, argv []string) (result []string) {
	if len(argv) == 0 {
		return []string{"#konnte rechnerliste nicht zusammenstellen"}
	} else {
		layoutNames := runSystemCommand([]string{"ls", "baselayouts/"}, RENDERFARMPATH)
		fmt.Printf("\n%d baselayouts gefunden", layoutNames)
		result = getRidOfDummies(strings.Split(layoutNames, "\n", 0))
	}
	return
}
func callGetClients(argc int, argv []string) (result []string) {
	if len(argv) == 0 {
		return []string{"#konnte rechnerliste nicht zusammenstellen"}
	} else {
		hostnameFiles := runSystemCommand([]string{"find", ".", "-name", "hostname"}, RENDERFARMPATH)
		catCommandStr := "cat\n" + hostnameFiles
		catCommand := strings.Split(catCommandStr, "\n", 0)
		catCommand = catCommand[0:(len(catCommand) - 1)]
		for i, k := range catCommand {
			fmt.Printf("\ncatCommand[%d]\"%s\"\n", i, k)
		}
		hostnames := runSystemCommand(catCommand, RENDERFARMPATH)
		hostnames = hostnames[0:(len(hostnames) - 1)]
		result = getRidOfDummies(strings.Split(hostnames, "\n", 0))
	}
	return
}

func runSystemCommand(argv []string, dir string) string {
	lookedPath, _ := exec.LookPath(argv[0])
	r, w, _ := os.Pipe()
	pid, _ := os.ForkExec(lookedPath, argv, nil, dir, []*os.File{nil, w, w})
	w.Close()
	os.Wait(pid, 0)
	var b bytes.Buffer
	io.Copy(&b, r)
	return b.String()
}

func isWhite(b byte) bool { return b == ' ' || b == '\n' || b == '\t' }

func myTrim(s string) string {
        var i, j int
        for i = 0; i < len(s) && isWhite(s[i]); i++ { }
        for j = len(s) - 1; j > 0 && isWhite(s[j]); j-- { }
        return s[i : j+1]
}

func getInfoByRechnername(rechnerName string) []string {
	fmt.Printf("---\n")
	// ls get mac,name,ip
	output := runSystemCommand(
		[]string{"grep", "-RB1", rechnerName, "."},
		CONFIGPATH)
	fmt.Printf(output)
	if len(myTrim(output)) == 0 {
		return []string{"#Not a valid name;"}
	}
	fmt.Printf("---\n")
	lsLines := strings.Split(output, "\n", 0)
	fmt.Printf("\"%s\"\n", lsLines[0])
//	fmt.Printf("\"%s\"\n", lsLines[1])
	lsReader0 := strings.NewReader(lsLines[0])
	//lsReader1 := strings.NewReader(lsLines[1])
	var macBuf, macPureBuf bytes.Buffer
	for i := 0; i < len(lsLines[0]); i++ {
		ch, err := lsReader0.ReadByte()
		if err != nil {
			break
		}
		if i < 2 {
			continue
		}
		if (i >= 2) && (i < 14) {
			macBuf.WriteByte(ch)
			macPureBuf.WriteByte(ch)
			switch i {
			case 3, 5, 7, 9, 11:
				macBuf.WriteByte(':')
			}
		}
	}
	macString := macBuf.String()
	macPureString := macPureBuf.String()
	// mac fertig
	preIP := strings.Split(lsLines[0], "-", 0)
	ipString := preIP[1]
	// ip fertig
	preName := strings.Split(lsLines[1], ":", 0)
	nameString := preName[1]
	// name fertig
	subnetString, gatewayString, dnsString := getSubnetGatewayDns(macPureString)
	// subnet,gw,dns fertig
	imageString, _ := getImageSwap(macPureString)
	// image,swap
	/*
		fmt.Printf("MAC-Adresse: %s\n", macString)
		fmt.Printf("MAC-Adresse(ohne): %s\n", macPureString)
		fmt.Printf("IP-Adresse: %s\n", ipString)
		fmt.Printf("Rechnername: %s\n", nameString)
		fmt.Printf("Subnetmask: %s\n", subnetString)
		fmt.Printf("Gateway: %s\n", gatewayString)
		fmt.Printf("Nameserver: %s\n", dnsString)
		fmt.Printf("Imagename: %s\n", imageString)
		fmt.Printf("Swapsize: %s\n", swapString)
	*/
	result := []string{nameString,
		ipString,
		subnetString,
		gatewayString,
		dnsString,
		imageString,
		macString,
		macPureString}
	return result
}

func getSubnetGatewayDns(macPureString string) (subnetString, gatewayString, dnsString string) {
	//get subnet-mask
	output := runSystemCommand(
		[]string{"grep", "netmask", "etc/network/interfaces"},
		RENDERFARMPATH+"overlays/"+macPureString)
	subnetFields := strings.Fields(output)
	subnetString = subnetFields[1]
	//get gateway
	output = runSystemCommand(
		[]string{"grep", "gateway", "etc/network/interfaces"},
		RENDERFARMPATH+"overlays/"+macPureString)
	gatewayFields := strings.Fields(output)
	gatewayString = gatewayFields[1]
	//get dns
	output = runSystemCommand(
		[]string{"grep", "dns-nameserver", "etc/network/interfaces"},
		RENDERFARMPATH+"overlays/"+macPureString)
	dnsFields := strings.Fields(output)
	dnsString = dnsFields[1]

	return
}

func getImageSwap(macPureString string) (imageString, swapString string) {
	//get image
	output := runSystemCommand(
		[]string{"cat", "config"},
		RENDERFARMPATH+"configs/"+macPureString)
	imageFields := strings.Fields(output)
	imageString = imageFields[0]
	swapString = imageFields[1]

	return
}
