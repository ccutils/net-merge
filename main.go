package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: <merge|test> [options]")
		os.Exit(1)
	}
	action := os.Args[1]
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)

	switch action {
	case "merge":
		merge()
	case "test":
		test()
	default:
		fmt.Println("Unknown action:", action)
	}
}

func merge() {
	out := flag.String("out", "merge.list", "Output file")
	flag.StringVar(out, "o", "merge.list", "Output file (shorthand)")

	typeFmt := flag.String("type", "txt", "Output type: txt or nft")
	flag.StringVar(typeFmt, "t", "txt", "Output type (shorthand)")

	name := flag.String("name", "netlist", "Name for nft define")
	flag.StringVar(name, "n", "netlist", "Name for nft define (shorthand)")

	urls := flag.String("url", "", "Comma-separated list of URLs")
	flag.StringVar(urls, "u", "", "Comma-separated list of URLs (shorthand)")

	files := flag.String("file", "", "Comma-separated list of local file paths")
	flag.StringVar(files, "f", "", "Comma-separated list of files (shorthand)")

	networks := flag.String("network", "", "Comma-separated CIDR networks")
	flag.StringVar(networks, "net", "", "Comma-separated CIDR networks (shorthand)")

	flag.Parse()

	allCIDRs := make(map[string]struct{})

	if *urls != "" {
		for _, url := range strings.Split(*urls, ",") {
			lines, err := readFromURL(url)
			if err != nil {
				fmt.Println("Error reading URL:", err)
				continue
			}
			for _, cidr := range lines {
				if isValidCIDR(cidr) {
					allCIDRs[cidr] = struct{}{}
				}
			}
		}
	}

	if *files != "" {
		for _, path := range strings.Split(*files, ",") {
			lines, err := readFromFile(path)
			if err != nil {
				fmt.Println("Error reading file:", err)
				continue
			}
			for _, cidr := range lines {
				if isValidCIDR(cidr) {
					allCIDRs[cidr] = struct{}{}
				}
			}
		}
	}

	if *networks != "" {
		for _, cidr := range strings.Split(*networks, ",") {
			if isValidCIDR(cidr) {
				allCIDRs[cidr] = struct{}{}
			}
		}
	}

	var cidrList []*net.IPNet
	for k := range allCIDRs {
		_, netw, _ := net.ParseCIDR(k)
		cidrList = append(cidrList, netw)
	}

	merged := recursiveMergeCIDRs(cidrList)
	sort.Slice(merged, func(i, j int) bool {
		return compareIP(merged[i].IP, merged[j].IP) < 0
	})

	f, err := os.Create(*out)
	if err != nil {
		fmt.Println("Failed to create output file:", err)
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if *typeFmt == "nft" {
		fmt.Fprintf(w, "define %s = {\n", *name)
		for i, netw := range merged {
			sep := ","
			if i == len(merged)-1 {
				sep = ""
			}
			fmt.Fprintf(w, "    %s%s\n", netw.String(), sep)
		}
		fmt.Fprintln(w, "}")
	} else {
		for _, netw := range merged {
			fmt.Fprintln(w, netw.String())
		}
	}
	w.Flush()
}

func test() {
	in := flag.String("in", "merge.list", "Input file")
	flag.StringVar(in, "i", "merge.list", "Input file (shorthand)")

	typeFmt := flag.String("type", "txt", "Input type: txt or nft")
	flag.StringVar(typeFmt, "t", "txt", "Input type (shorthand)")

	name := flag.String("name", "netlist", "Name for nft define")
	flag.StringVar(name, "n", "netlist", "Name for nft define (shorthand)")

	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Please specify an IP address to test")
		return
	}
	testIP := args[0]
	ip := net.ParseIP(testIP)
	if ip == nil {
		fmt.Println("Invalid IP address")
		return
	}

	lines, err := readFromFile(*in)
	if err != nil {
		fmt.Println("Failed to read file:", err)
		return
	}

	var cidrs []string
	if *typeFmt == "nft" {
		start := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "define "+*name) {
				start = true
				continue
			}
			if start {
				if line == "}" {
					break
				}
				cidr := strings.TrimRight(strings.TrimSpace(line), ",")
				cidrs = append(cidrs, cidr)
			}
		}
	} else {
		cidrs = lines
	}

	for _, cidr := range cidrs {
		if _, network, err := net.ParseCIDR(cidr); err == nil {
			if network.Contains(ip) {
				fmt.Println("IP is in:", cidr)
				return
			}
		}
	}
	fmt.Println("IP not found in any CIDR range")
}

func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(strings.TrimSpace(cidr))
	return err == nil
}

func readFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readLines(f), nil
}

func readFromURL(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New("non-200 response")
	}
	return readLines(resp.Body), nil
}

func readLines(r io.Reader) []string {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func compareIP(a, b net.IP) int {
	a = a.To4()
	b = b.To4()
	return bytes.Compare(a, b)
}

func mergeCIDRs(cidrs []*net.IPNet) []*net.IPNet {
	sort.Slice(cidrs, func(i, j int) bool {
		return compareIP(cidrs[i].IP, cidrs[j].IP) < 0
	})
	result := []*net.IPNet{}
	for _, cidr := range cidrs {
		inserted := false
		for i, existing := range result {
			if canMerge(existing, cidr) {
				merged := mergeTwo(existing, cidr)
				result[i] = merged
				inserted = true
				break
			}
		}
		if !inserted {
			result = append(result, cidr)
		}
	}
	return result
}

func recursiveMergeCIDRs(cidrs []*net.IPNet) []*net.IPNet {
	prevLen := -1
	for prevLen != len(cidrs) {
		prevLen = len(cidrs)
		cidrs = mergeCIDRs(cidrs)
	}
	return cidrs
}

func canMerge(a, b *net.IPNet) bool {
	onesA, bitsA := a.Mask.Size()
	onesB, bitsB := b.Mask.Size()
	if onesA != onesB || bitsA != 32 || bitsB != 32 {
		return false
	}
	size := 1 << (32 - onesA)
	ipA := ipToInt(a.IP)
	ipB := ipToInt(b.IP)
	return ipA+uint32(size) == ipB || ipB+uint32(size) == ipA
}

func mergeTwo(a, b *net.IPNet) *net.IPNet {
	prefix, _ := a.Mask.Size()
	if prefix == 0 {
		return a
	}
	newPrefix := prefix - 1
	newMask := net.CIDRMask(newPrefix, 32)
	base := a.IP.Mask(newMask)
	return &net.IPNet{IP: base, Mask: newMask}
}

func ipToInt(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}
