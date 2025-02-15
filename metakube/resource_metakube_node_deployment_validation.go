package metakube

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

func validateNodeSpecMatchesCluster() schema.CustomizeDiffFunc {
	return func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
		k := meta.(*metakubeProviderMeta)
		clusterID := d.Get("cluster_id").(string)
		if clusterID == "" {
			return nil
		}
		projectID := d.Get("project_id").(string)
		if projectID == "" {
			return nil
		}
		cluster, err := metakubeGetCluster(ctx, projectID, clusterID, k)
		if err != nil {
			return err
		}
		clusterProvider, err := getClusterCloudProvider(cluster)
		if err != nil {
			return err
		}
		err = validateProviderMatchesCluster(d, clusterProvider)
		if err != nil {
			return err
		}
		return nil
	}
}

func getClusterCloudProvider(c *models.Cluster) (string, error) {
	switch {
	case c.Spec.Cloud.Bringyourown != nil:
		return "bringyourown", nil
	case c.Spec.Cloud.Aws != nil:
		return "aws", nil
	case c.Spec.Cloud.Openstack != nil:
		return "openstack", nil
	case c.Spec.Cloud.Azure != nil:
		return "azure", nil
	default:
		return "", fmt.Errorf("could not find cloud provider for cluster")

	}
}

func validateProviderMatchesCluster(d *schema.ResourceDiff, clusterProvider string) error {
	var availableProviders = []string{"bringyourown", "aws", "openstack", "azure"}
	var provider string

	for _, p := range availableProviders {
		providerField := fmt.Sprintf("spec.0.template.0.cloud.0.%s", p)
		_, ok := d.GetOk(providerField)
		if ok {
			provider = p
			break
		}
	}
	if provider != clusterProvider {
		return fmt.Errorf("provider for node deployment must (%s) match cluster provider (%s)", provider, clusterProvider)
	}
	return nil

}

func metakubeGetCluster(ctx context.Context, proj, cls string, k *metakubeProviderMeta) (*models.Cluster, error) {
	p := project.NewGetClusterV2Params().
		WithContext(ctx).
		WithProjectID(proj).
		WithClusterID(cls)
	r, err := k.client.Project.GetClusterV2(p, k.auth)
	if err != nil {
		return nil, fmt.Errorf("unable to get cluster %s in project %s - error: %v", cls, proj, err)
	}

	return r.Payload, nil
}

func validateAutoscalerFields() schema.CustomizeDiffFunc {
	return func(ctx context.Context, d *schema.ResourceDiff, _ interface{}) error {
		minReplicas, ok1 := d.GetOk("spec.0.min_replicas")
		maxReplicas, ok2 := d.GetOk("spec.0.max_replicas")
		if ok1 && ok1 != ok2 {
			return fmt.Errorf("to configure autoscaler both min_replicas and max_replicas must be set")
		}
		if ok1 == false {
			return nil
		}

		if minReplicas.(int) > maxReplicas.(int) {
			return fmt.Errorf("min_replicas must be smaller than max_replicas")
		}

		replicas := 1
		if v, ok := d.GetOk("spec.0.replicas"); ok {
			replicas = v.(int)
		}
		if replicas > maxReplicas.(int) {
			return fmt.Errorf("max_replicas can't be smaller than replicas")
		}
		if replicas < minReplicas.(int) {
			return fmt.Errorf("min_replicas can't be bigger than replicas")
		}
		return nil
	}
}
