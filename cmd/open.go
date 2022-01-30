package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/kevinburke/ssh_config"

	"github.com/spf13/cobra"
)

var bastionName string
var clusterName string
var daemon bool

// openCmd represents the open command
var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open an SSHuttle tunnel to a cluster.",
	Long:  `Open an SSHuttle tunnel to a cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		daemonMode := "disabled"
		pidFile := "/tmp/tunnel.pid"

		if daemon {
			daemonMode = "enabled"
			_, err := os.Stat(pidFile)
			if err == nil {
				log.Fatal("tunnel already opened in the background, exiting")
			}
		}

		log.Printf(`parameters : 
bastion instance name : %s
cluster name : %s
daemon mode : %s`, bastionName, clusterName, daemonMode)

		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-3"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}

		ec2Client := ec2.NewFromConfig(cfg)

		filters := []types.Filter{{Name: aws.String("tag:Name"), Values: []string{bastionName}}, {Name: aws.String("instance-state-name"), Values: []string{"running"}}}

		input := ec2.DescribeInstancesInput{Filters: filters}

		ec2result, err := ec2Client.DescribeInstances(context.TODO(), &input)
		if err != nil {
			log.Fatalf("failed to describe instances, %v", err)
		}

		if len(ec2result.Reservations) == 0 {
			log.Fatalf("the bastion instance \"%s\" does not exist", bastionName)
		}

		bastionInstanceId := *ec2result.Reservations[0].Instances[0].InstanceId

		log.Printf("found instance id for %s : %s", bastionName, bastionInstanceId)

		eksClient := eks.NewFromConfig(cfg)

		eksResult, err := eksClient.DescribeCluster(context.TODO(), &eks.DescribeClusterInput{Name: aws.String(clusterName)})
		if err != nil {
			log.Fatalf("failed to describe cluster, %v", err)
		}

		if eksResult.Cluster.Endpoint == nil {
			log.Fatalf("the cluster \"%s\" does not exist", clusterName)
		}

		eksApiEndpoint := strings.Replace(*eksResult.Cluster.Endpoint, "https://", "", 1)

		log.Printf("eks api endpoint : %s", eksApiEndpoint)

		ips, err := net.LookupIP(eksApiEndpoint)
		if err != nil {
			log.Fatalf("failed to get cluster endpoint ip, %v", err)
		}

		ipString := ""

		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				ipString += ipv4.String() + "/32 "
			}
		}

		log.Printf("cluster api endpoint ips : %s", ipString)

		f, err := os.OpenFile(filepath.Join(os.Getenv("HOME"), ".ssh", "config"), os.O_RDWR, os.ModePerm)
		if err != nil {
			log.Fatalf("failed to get ssh config, %v", err)
		}

		sshConfig, err := ssh_config.Decode(f)
		if err != nil {
			log.Fatalf("failed to get ssh config, %v", err)
		}
		f.Close()

		bastionConfigString := `
Host i-*
  ProxyCommand bash -c "aws ssm start-session --target %h --document-name AWS-StartSSHSession --parameters 'portNumber=%p'"`

		bastionConfig, err := ssh_config.Decode(strings.NewReader(bastionConfigString))
		if err != nil {
			log.Fatalf("failed to read bastion config string, %v", err)
		}

		containsPattern := false
		testPattern, _ := ssh_config.NewPattern("i-*")

		for _, host := range sshConfig.Hosts {
			for _, pattern := range host.Patterns {
				if pattern == testPattern {
					containsPattern = true
				}
			}
		}

		if containsPattern {
			sshConfig.Hosts = append(sshConfig.Hosts, bastionConfig.Hosts...)
		}

		f2, err := os.OpenFile(filepath.Join(os.Getenv("HOME"), ".ssh", "config"), os.O_RDWR|os.O_TRUNC, os.ModePerm)
		if err != nil {
			log.Fatalf("failed to open ssh config, %v", err)
		}

		_, err = f2.WriteString(sshConfig.String())
		if err != nil {
			log.Fatalf("failed to write ssh config, %v", err)
		}

		f2.Sync()

		log.Printf("opening tunnel")

		commandString := "sshuttle -r ubuntu@" + bastionInstanceId + " " + ipString
		if daemon {
			commandString = "sshuttle --daemon --pidfile=" + pidFile + " -r ubuntu@" + bastionInstanceId + " " + ipString
		}

		command := exec.Command("bash", "-c", commandString)
		output, err := command.CombinedOutput()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + string(output))
			return
		}

		if daemon {
			log.Printf("connected (pidfile : %s)", pidFile)
		}
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
	openCmd.Flags().StringVarP(&bastionName, "bastion", "b", "bastion", "Name of the bastion instance to connect to")
	openCmd.Flags().StringVarP(&clusterName, "cluster", "c", "eks_cluster", "Name of the EKS cluster to connect to")
	openCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Run in the background")
}
