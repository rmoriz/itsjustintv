package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/rmoriz/itsjustintv/internal/twitch"
)

var subscriptionsCmd = &cobra.Command{
	Use:   "subscriptions",
	Short: "Manage Twitch EventSub subscriptions",
	Long:  `Commands to list, create, and delete Twitch EventSub subscriptions.`,
}

var listSubscriptionsCmd = &cobra.Command{
	Use:   "list",
	Short: "List current Twitch EventSub subscriptions",
	Long:  `List all current EventSub subscriptions for your Twitch application.`,
	RunE:  runListSubscriptions,
}

var syncSubscriptionsCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync subscriptions with configuration",
	Long:  `Create missing subscriptions based on your configuration and remove unwanted ones.`,
	RunE:  runSyncSubscriptions,
}

func init() {
	rootCmd.AddCommand(subscriptionsCmd)
	subscriptionsCmd.AddCommand(listSubscriptionsCmd)
	subscriptionsCmd.AddCommand(syncSubscriptionsCmd)
}

func runListSubscriptions(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Setup logger
	logger := setupLogger(verbose)

	// Create Twitch client
	client := twitch.NewClient(cfg, logger)
	if err := client.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start Twitch client: %w", err)
	}
	defer client.Stop()

	// Create subscription manager
	subManager := twitch.NewSubscriptionManager(cfg, logger, client)

	// Get subscriptions
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subs, err := subManager.GetSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}

	// Display results
	fmt.Printf("EventSub Subscriptions Summary:\n")
	fmt.Printf("Total subscriptions: %d\n", subs.Total)
	fmt.Printf("Total cost: %d\n", subs.TotalCost)
	fmt.Printf("Max total cost: %d\n\n", subs.MaxTotalCost)

	if len(subs.Data) == 0 {
		fmt.Println("No subscriptions found.")
		return nil
	}

	fmt.Printf("%-20s %-15s %-20s %-15s %-20s\n", "ID", "Type", "Status", "Broadcaster ID", "Created At")
	fmt.Println("--------------------------------------------------------------------------------------------")

	for _, sub := range subs.Data {
		broadcasterID := "N/A"
		if bid, ok := sub.Condition["broadcaster_user_id"].(string); ok {
			broadcasterID = bid
		}

		fmt.Printf("%-20s %-15s %-20s %-15s %-20s\n",
			sub.ID[:8]+"...",
			sub.Type,
			sub.Status,
			broadcasterID,
			sub.CreatedAt.Format("2006-01-02 15:04"))
	}

	return nil
}

func runSyncSubscriptions(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Setup logger
	logger := setupLogger(verbose)

	// Create Twitch client
	client := twitch.NewClient(cfg, logger)
	if err := client.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start Twitch client: %w", err)
	}
	defer client.Stop()

	// Resolve missing user IDs for streamers
	ctx := context.Background()
	if err := config.ResolveStreamerUserIDs(ctx, cfg, client); err != nil {
		logger.Warn("Failed to resolve some streamer user IDs", "error", err)
	}

	// Create subscription manager
	subManager := twitch.NewSubscriptionManager(cfg, logger, client)

	// Sync subscriptions
	syncCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := subManager.SyncSubscriptions(syncCtx); err != nil {
		return fmt.Errorf("failed to sync subscriptions: %w", err)
	}

	fmt.Println("Subscription sync completed successfully!")
	return nil
}