// redis contains utils to configure redis clusters
package redis

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

const (
	defaultPort     = 6379
	defaultPassword = ""
	slotLength      = 16385
)

// ConfigureClusterMode joins all nodes in the same cluster.
// To join modes I use cluster meet.
func ConfigureClusterMode(ctx context.Context, ips []string) error {
	if len(ips) == 0 {
		return nil
	}
	// I need to meet them: A --> B, B --> C, C --> D. (Gossip does the rest of the job)
	// TODO: Test when deleting all pods at time.
	previous := ips[0]
	for _, next := range ips[1:] {
		c := getConection(previous)
		err := c.ClusterMeet(ctx, next, strconv.Itoa(defaultPort)).Err()
		previous = next
		if err != nil {
			return err
		}
	}
	return nil
}

func ConfigureSlots(ctx context.Context, ips []string) error {
	// Are all slots assigned?
	if len(ips) == 0 {
		return nil
	}
	ok, err := areSlotsWellAssigned(ctx, ips)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	err = deleteSlots(ctx, ips) // TODO: Change this for export/migrate slots...
	if err != nil {
		return err
	}
	// c.ClusterAddSlotsRange(ctx context.Context, min int, max int)
	// c.ClusterDelSlots(ctx context.Context, slots ...int)
	// Bootstrap masters.
	for index, ip := range ips {
		first, last := calculateSlotRange(index, len(ips))
		c := getConection(ip)
		err := c.ClusterAddSlotsRange(ctx, first, last).Err()
		if err != nil {
			return err
		}
	}
	return nil

}

func areSlotsWellAssigned(ctx context.Context, ips []string) (bool, error) {
	if len(ips) == 0 {
		return true, nil
	}
	c := getConection(ips[0])
	clusterSlots, err := c.ClusterSlots(ctx).Result()
	if err != nil {
		return false, err
	}
	for i, s := range clusterSlots {
		start, end := calculateSlotRange(i, len(ips))
		if !(start == s.Start && end == s.End) {
			return false, nil
		}
	}
	return true, nil
}

func deleteSlots(ctx context.Context, ips []string) error {
	if len(ips) == 0 {
		return nil
	}
	c := getConection(ips[0])
	clusterSlots, err := c.ClusterSlots(ctx).Result()
	if err != nil {
		return err
	}
	for _, s := range clusterSlots {
		for _, n := range s.Nodes {
			ip := strings.Split(n.Addr, ":")[0]
			c := getConection(ip)
			err = c.ClusterDelSlotsRange(ctx, s.Start, s.End).Err()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/*
func DeleteSlots(ctx context.Context, ips []string) error {
	// Are all slots assigned?
	if len(ips) == 0 {
		return nil
	}
	// c.ClusterAddSlotsRange(ctx context.Context, min int, max int)
	// c.ClusterDelSlots(ctx context.Context, slots ...int)
	// Bootstrap masters.
	for _, ip := range ips {
		c := getConection(ip)
		err := c.ClusterDelSlotsRange(ctx context.Context, min int, max int)
		if err != nil {
			return err
		}
	}
	return nil

}
*/

func calculateSlotRange(index, nodes int) (first, last int) {
	slotsByNode := slotLength / nodes
	first = index * slotsByNode
	last = (index + 1) * slotsByNode
	if index == nodes-1 {
		last += slotLength % nodes
	}
	return first, last
}

func getConection(ip string) *redis.Client {
	addr := ip + ":" + strconv.Itoa(defaultPort)
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: defaultPassword,
		DB:       0, // use default DB
	})
}
