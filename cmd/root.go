package cmd

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Open / close a SSHuttle tunnel to an EKS cluster.",
	Long: `Open / close a SSHuttle tunnel to an EKS cluster.

Prerequisites :
- An EKS cluster with private access enabled
- An Ubuntu EC2 instance
 - in the same subnet as the EKS cluster,
 - running the SSM agent,
 - associated to a public SSH key that you own
	
Examples: 
- ./tunnel (interactive mode)
- ./tunnel open -b my_bastion -c my_cluster (open a tunnel through a bastion named "my_bastion" to a cluster named "my_cluster")
- ./tunnel close (close a tunnel opened in the background)
	`,
	Run: func(cmd *cobra.Command, args []string) {
		commandPrompt := promptui.Select{
			Label: "Command",
			Items: []string{"open", "close"},
		}

		_, result, err := commandPrompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		if result == "open" {
			bastionPrompt := promptui.Prompt{
				Label: "Bastion name (default : bastion)",
			}

			result, err := bastionPrompt.Run()

			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return
			}

			if result != "" {
				bastionName = result
			}

			clusterPrompt := promptui.Prompt{
				Label: "Cluster name (default : eks_cluster)",
			}

			result, err = clusterPrompt.Run()

			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return
			}

			if result != "" {
				clusterName = result
			}

			daemonPrompt := promptui.Select{
				Label: "Daemon mode",
				Items: []string{"no", "yes"},
			}

			_, result, err = daemonPrompt.Run()

			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return
			}

			if result == "yes" {
				daemon = true
			}

			openCmd.Run(cmd, []string{})
		} else {
			closeCmd.Run(cmd, []string{})
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {}
