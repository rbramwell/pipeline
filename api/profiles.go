package api

import (
	"io/ioutil"
	"net/http"

	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/model/defaults"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	acsk2 "github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	aks2 "github.com/banzaicloud/pipeline/pkg/cluster/aks"
	ec22 "github.com/banzaicloud/pipeline/pkg/cluster/ec2"
	eks2 "github.com/banzaicloud/pipeline/pkg/cluster/eks"
	gke2 "github.com/banzaicloud/pipeline/pkg/cluster/gke"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers"
	oke2 "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	distributionTypeKey = "distribution"
	nameKey             = "name"
)

// GetClusterProfiles handles /profiles/cluster/:type GET api endpoint.
// Sends back the saved cluster profiles
func GetClusterProfiles(c *gin.Context) {

	distributionType := c.Param(distributionTypeKey)
	//log.Infof("Start getting saved cluster profiles [%s]", distributionType)
	//
	//resp, err := getProfiles(distributionType)
	//if err != nil {
	//	log.Errorf("Error during getting defaults to %s: %s", distributionType, err.Error())
	//	c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
	//		Code:    http.StatusBadRequest,
	//		Message: err.Error(),
	//		Error:   err.Error(),
	//	})
	//} else {
	//	c.JSON(http.StatusOK, resp)
	//}

	profile, err := getDefaultProfile(distributionType)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, profile)

}

func getDefaultProfile(distributionType string) (*pkgCluster.CreateClusterRequest, error) {
	defaults, images, err := readFiles()
	if err != nil {
		return nil, err
	}

	switch distributionType {
	case pkgCluster.ACSK:
		return createACSKRequest(&defaults.Distributions.ACSK, defaults.DefaultNodePoolName), nil
	case pkgCluster.AKS:
		return createAKSRequest(&defaults.Distributions.AKS, defaults.DefaultNodePoolName), nil
	case pkgCluster.EC2:
		return createEC2Request(&defaults.Distributions.EC2, defaults.DefaultNodePoolName, images), nil
	case pkgCluster.EKS:
		return createEKSRequest(&defaults.Distributions.EKS, defaults.DefaultNodePoolName, images), nil
	case pkgCluster.GKE:
		return createGKERequest(&defaults.Distributions.GKE, defaults.DefaultNodePoolName), nil
	case pkgCluster.OKE:
		return createOKERequest(&defaults.Distributions.OKE, defaults.DefaultNodePoolName), nil

	}

	return nil, errors.New("not supported distribution")
}

func createOKERequest(oke *DefaultsOKE, defaultNodePoolName string) *pkgCluster.CreateClusterRequest {

	nodepools := make(map[string]*oke2.NodePool)
	nodepools[defaultNodePoolName] = &oke2.NodePool{
		Version: oke.NodePools.Version,
		Count:   uint(oke.NodePools.Count),
		Image:   oke.NodePools.Image,
		Shape:   oke.NodePools.Shape,
	}

	return &pkgCluster.CreateClusterRequest{
		Location: oke.Location,
		Cloud:    pkgCluster.Oracle,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterOKE: &oke2.Cluster{
				Version:   oke.Version,
				NodePools: nodepools,
			},
		},
	}
}

func createGKERequest(gke *DefaultsGKE, defaultNodePoolName string) *pkgCluster.CreateClusterRequest {

	nodepools := make(map[string]*gke2.NodePool)
	nodepools[defaultNodePoolName] = &gke2.NodePool{
		Autoscaling:      gke.NodePools.Autoscaling,
		MinCount:         gke.NodePools.MinCount,
		MaxCount:         gke.NodePools.MaxCount,
		Count:            gke.NodePools.Count,
		NodeInstanceType: gke.NodePools.InstanceType,
	}

	return &pkgCluster.CreateClusterRequest{
		Location: gke.Location,
		Cloud:    pkgCluster.Google,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterGKE: &gke2.CreateClusterGKE{
				NodeVersion: gke.NodeVersion,
				NodePools:   nodepools,
				Master: &gke2.Master{
					Version: gke.MasterVersion,
				},
			},
		},
	}

}

func createEKSRequest(eks *DefaultsEKS, defaultNodePoolName string, images DefaultAmazonImages) *pkgCluster.CreateClusterRequest {

	image := getAmazonImage(images.EKS, eks.Location)

	nodepools := make(map[string]*ec22.NodePool)
	nodepools[defaultNodePoolName] = &ec22.NodePool{
		InstanceType: eks.NodePools.InstanceType,
		SpotPrice:    eks.NodePools.SpotPrice,
		Autoscaling:  eks.NodePools.Autoscaling,
		MinCount:     eks.NodePools.MinCount,
		MaxCount:     eks.NodePools.MaxCount,
		Count:        eks.NodePools.Count,
		Image:        image,
	}

	return &pkgCluster.CreateClusterRequest{
		Location: eks.Location,
		Cloud:    pkgCluster.Amazon,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterEKS: &eks2.CreateClusterEKS{
				Version:   eks.Version,
				NodePools: nodepools,
			},
		},
	}

}

func createEC2Request(ec2 *DefaultsEC2, defaultNodePoolName string, images DefaultAmazonImages) *pkgCluster.CreateClusterRequest {

	image := getAmazonImage(images.EC2, ec2.Location)

	nodepools := make(map[string]*ec22.NodePool)
	nodepools[defaultNodePoolName] = &ec22.NodePool{
		InstanceType: ec2.NodePools.InstanceType,
		SpotPrice:    ec2.NodePools.SpotPrice,
		Autoscaling:  ec2.NodePools.Autoscaling,
		MinCount:     ec2.NodePools.MinCount,
		MaxCount:     ec2.NodePools.MaxCount,
		Count:        ec2.NodePools.Count,
		Image:        image,
	}

	return &pkgCluster.CreateClusterRequest{
		Location: ec2.Location,
		Cloud:    pkgCluster.Amazon,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterEC2: &ec22.CreateClusterEC2{
				NodePools: nodepools,
				Master: &ec22.CreateAmazonMaster{
					InstanceType: ec2.MasterInstanceType,
					Image:        image,
				},
			},
		},
	}
}

func getAmazonImage(images AmazonImages, location string) string {
	return images[location]
}

func createAKSRequest(aks *DefaultsAKS, defaultNodePoolName string) *pkgCluster.CreateClusterRequest {

	nodepool := make(map[string]*aks2.NodePoolCreate)
	nodepool[defaultNodePoolName] = &aks2.NodePoolCreate{
		Autoscaling:      aks.NodePools.Autoscaling,
		MinCount:         aks.NodePools.MinCount,
		MaxCount:         aks.NodePools.MaxCount,
		Count:            aks.NodePools.Count,
		NodeInstanceType: aks.NodePools.InstanceType,
	}

	return &pkgCluster.CreateClusterRequest{
		Location: aks.Location,
		Cloud:    pkgCluster.Azure,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterAKS: &aks2.CreateClusterAKS{
				KubernetesVersion: aks.Version,
				NodePools:         nodepool,
			},
		},
	}
}

func createACSKRequest(acsk *DefaultsACSK, defaultNodePoolName string) *pkgCluster.CreateClusterRequest {
	nodepools := make(acsk2.NodePools)
	nodepools[defaultNodePoolName] = &acsk2.NodePool{
		InstanceType:       acsk.NodePools.InstanceType,
		SystemDiskCategory: acsk.NodePools.SystemDiskCategory,
		//SystemDiskSize:     acsk.NodePools.SystemDiskSize,  // todo missing
		//LoginPassword:      acsk.NodePools.LoginPassword,  // todo missing
		Count: int(acsk.NodePools.Count),
		Image: acsk.NodePools.Image,
	}

	return &pkgCluster.CreateClusterRequest{
		Location: acsk.Location,
		Cloud:    providers.Alibaba,
		Properties: &pkgCluster.CreateClusterProperties{
			CreateClusterACSK: &acsk2.CreateClusterACSK{
				RegionID:                 acsk.RegionId,
				ZoneID:                   acsk.ZoneId,
				MasterInstanceType:       acsk.MasterInstanceType,
				MasterSystemDiskCategory: acsk.MasterSystemDiskCategory,
				//MasterSystemDiskSize:     acsk.MasterSystemDiskSize, // todo missing
				//KeyPair:                  acsk.KeyPair, // todo missing
				NodePools: nodepools,
			},
		},
	}
}

func readFiles() (defaults Defaults, images DefaultAmazonImages, err error) {

	if err = readYaml("defaults/defaults.yaml", &defaults); err != nil {
		return
	}

	err = readYaml("defaults/defaults-amazon-images.yaml", &images)

	return
}

func readYaml(filePath string, out interface{}) error {
	f, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(f, out)
	if err != nil {
		return err
	}

	return nil
}

type DefaultAmazonImages struct {
	EC2 AmazonImages `yaml:"ec2"`
	EKS AmazonImages `yaml:"eks"`
}

type AmazonImages map[string]string

type Defaults struct {
	DefaultNodePoolName string               `yaml:"defaultNodePoolName"`
	Distributions       DefaultsDistribution `yaml:"distributions"`
}

type DefaultsDistribution struct {
	ACSK DefaultsACSK `yaml:"acsk"`
	AKS  DefaultsAKS  `yaml:"aks"`
	EC2  DefaultsEC2  `yaml:"ec2"`
	EKS  DefaultsEKS  `yaml:"eks"`
	GKE  DefaultsGKE  `yaml:"gke"`
	OKE  DefaultsOKE  `yaml:"oke"`
}

type DefaultsACSK struct {
	Location                 string                `yaml:"location"`
	RegionId                 string                `yaml:"regionId"`
	ZoneId                   string                `yaml:"zoneId"`
	MasterInstanceType       string                `yaml:"masterInstanceType"`
	MasterSystemDiskCategory string                `yaml:"masterSystemDiskCategory"`
	NodePools                DefaultsACSKNodePools `yaml:"nodePools"`
}

type DefaultsAKS struct {
	Location  string               `yaml:"location"`
	Version   string               `yaml:"version"`
	NodePools DefaultsAKSNodePools `yaml:"nodePools"`
}

type DefaultsEC2 struct {
	Location           string                  `yaml:"location"`
	MasterInstanceType string                  `yaml:"masterInstanceType"`
	NodePools          DefaultsAmazonNodePools `yaml:"nodePools"`
}

type DefaultsEKS struct {
	Location  string                  `yaml:"location"`
	Version   string                  `yaml:"version"`
	NodePools DefaultsAmazonNodePools `yaml:"nodePools"`
}

type DefaultsGKE struct {
	Location      string               `yaml:"location"`
	MasterVersion string               `yaml:"masterVersion"`
	NodeVersion   string               `yaml:"nodeVersion"`
	NodePools     DefaultsGKENodePools `yaml:"nodePools"`
}

type DefaultsOKE struct {
	Location  string               `yaml:"location"`
	Version   string               `yaml:"version"`
	NodePools DefaultsOKENodePools `yaml:"nodePools"`
}

type DefaultsACSKNodePools struct {
	Autoscaling        bool   `yaml:"autoscaling"`
	Count              int    `yaml:"count"`
	MinCount           int    `yaml:"minCount"`
	MaxCount           int    `yaml:"maxCount"`
	Image              string `yaml:"image"`
	InstanceType       string `yaml:"instanceType"`
	SystemDiskCategory string `yaml:"systemDiskCategory"`
}

type DefaultsAKSNodePools struct {
	Autoscaling  bool   `yaml:"autoscaling"`
	Count        int    `yaml:"count"`
	MinCount     int    `yaml:"minCount"`
	MaxCount     int    `yaml:"maxCount"`
	InstanceType string `yaml:"instanceType"`
}

type DefaultsAmazonNodePools struct {
	InstanceType string `yaml:"instanceType"`
	SpotPrice    string `yaml:"spotPrice"`
	Autoscaling  bool   `yaml:"autoscaling"`
	Count        int    `yaml:"count"`
	MinCount     int    `yaml:"minCount"`
	MaxCount     int    `yaml:"maxCount"`
}

type DefaultsGKENodePools struct {
	Autoscaling  bool   `yaml:"autoscaling"`
	Count        int    `yaml:"count"`
	MinCount     int    `yaml:"minCount"`
	MaxCount     int    `yaml:"maxCount"`
	InstanceType string `yaml:"instanceType"`
}

type DefaultsOKENodePools struct {
	Version  string `yaml:"version"`
	Count    int    `yaml:"count"`
	MinCount int    `yaml:"minCount"`
	MaxCount int    `yaml:"maxCount"`
	Image    string `yaml:"image"`
	Shape    string `yaml:"shape"`
}

// AddClusterProfile handles /profiles/cluster/:type POST api endpoint.
// Saves ClusterProfileRequest data into the database.
// Saving failed if profile with the given name is already exists
func AddClusterProfile(c *gin.Context) {

	log.Info("Start getting save cluster profile")

	log.Debug("Bind json into ClusterProfileRequest struct")
	// bind request body to struct
	var profileRequest pkgCluster.ClusterProfileRequest
	if err := c.BindJSON(&profileRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Parsing request succeeded")
	log.Infof("Convert ClusterProfileRequest into ClusterProfile model with name: %s", profileRequest.Name)

	// convert request into ClusterProfile model
	if prof, err := convertRequestToProfile(&profileRequest); err != nil {
		log.Error("Error during convert profile: &s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during convert profile",
			Error:   err.Error(),
		})
	} else if !prof.IsDefinedBefore() {
		// name is free
		log.Info("Convert succeeded")
		log.Info("Save cluster profile into database")
		if err := prof.SaveInstance(); err != nil {
			// save failed
			log.Errorf("Error during persist cluster profile: %s", err.Error())
			c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Error during persist cluster profile",
				Error:   err.Error(),
			})
		} else {
			// save succeeded
			log.Info("Save cluster profile succeeded")
			c.Status(http.StatusCreated)
		}
	} else {
		// profile with given name is already exists
		log.Error("Cluster profile with the given name is already exists")
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Cluster profile with the given name is already exists, please update not create profile",
			Error:   "Cluster profile with the given name is already exists, please update not create profile",
		})
	}

}

// getProfiles loads cluster profiles from database by distribution
func getProfiles(distribution string) ([]pkgCluster.ClusterProfileResponse, error) {

	var response []pkgCluster.ClusterProfileResponse
	profiles, err := defaults.GetAllProfiles(distribution)
	if err != nil {
		// error during getting profiles
		return nil, err
	}
	for _, p := range profiles {
		r := p.GetProfile()
		response = append(response, *r)
	}
	return response, nil

}

// convertRequestToProfile converts a ClusterProfileRequest into ClusterProfile
func convertRequestToProfile(request *pkgCluster.ClusterProfileRequest) (defaults.ClusterProfile, error) {

	switch request.Cloud {
	case pkgCluster.Amazon:
		if request.Properties.EC2 != nil {
			var ec2Profile defaults.EC2Profile
			ec2Profile.UpdateProfile(request, false)
			return &ec2Profile, nil
		}
		var eksProfile defaults.EKSProfile
		eksProfile.UpdateProfile(request, false)
		return &eksProfile, nil
	case pkgCluster.Azure:
		var aksProfile defaults.AKSProfile
		aksProfile.UpdateProfile(request, false)
		return &aksProfile, nil
	case pkgCluster.Google:
		var gkeProfile defaults.GKEProfile
		gkeProfile.UpdateProfile(request, false)
		return &gkeProfile, nil
	case pkgCluster.Oracle:
		var okeProfile oracle.Profile
		okeProfile.UpdateProfile(request, false)
		return &okeProfile, nil
	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}

}

// UpdateClusterProfile handles /cluster/profiles/:type PUT api endpoint.
// Updates existing cluster profiles.
// Updating failed if the name is the default name.
func UpdateClusterProfile(c *gin.Context) {

	log.Debug("Bind json into ClusterProfileRequest struct")
	// bind request body to struct
	var profileRequest pkgCluster.ClusterProfileRequest
	if err := c.BindJSON(&profileRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	log.Debug("Parsing request succeeded")

	if defaults.GetDefaultProfileName() == profileRequest.Name {
		// default profiles cannot updated
		log.Error("The default profile cannot be updated") // todo move to constants
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "The default profile cannot be updated",
			Error:   "The default profile cannot be updated",
		})
		return
	}

	log.Infof("Load cluster from database: %s[%s]", profileRequest.Name, profileRequest.Cloud)

	// load cluster profile from database
	if profile, err := defaults.GetProfile(profileRequest.Cloud, profileRequest.Name); err != nil {
		// load from db failed
		log.Error(errors.Wrap(err, "Error during getting profile"))
		sendBackGetProfileErrorResponse(c, err)
	} else if err := profile.UpdateProfile(&profileRequest, true); err != nil {
		// updating failed
		log.Error(errors.Wrap(err, "Error during update profile"))
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during update profile",
			Error:   err.Error(),
		})
	} else {
		// update success
		log.Infof("Update succeeded")
		c.Status(http.StatusCreated)
	}

}

// DeleteClusterProfile handles /cluster/profiles/:type/:name DELETE api endpoint.
// Deletes saved cluster profile.
// Deleting failed if the name is the default name.
func DeleteClusterProfile(c *gin.Context) {

	distribution := c.Param(distributionTypeKey)
	name := c.Param(nameKey)
	log.Infof("Start deleting cluster profile: %s[%s]", name, distribution)

	if defaults.GetDefaultProfileName() == name {
		// default profile cannot deleted
		log.Error("The default profile cannot be deleted")
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "The default profile cannot be deleted",
			Error:   "The default profile cannot be deleted",
		})
		return
	}

	log.Infof("Load cluster profile from database: %s[%s]", name, distribution)

	// load cluster profile from database
	if profile, err := defaults.GetProfile(distribution, name); err != nil {
		// load from database failed
		log.Error(errors.Wrap(err, "Error during getting profile"))
		sendBackGetProfileErrorResponse(c, err)
	} else {
		log.Info("Getting profile succeeded")
		log.Info("Delete from database")
		if err := profile.DeleteProfile(); err != nil {
			// delete from db failed
			log.Error(errors.Wrap(err, "Error during profile delete"))
			c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Error during profile delete",
				Error:   err.Error(),
			})
		} else {
			// delete succeeded
			log.Info("Delete from database succeeded")
			c.Status(http.StatusOK)
		}
	}

}

func sendBackGetProfileErrorResponse(c *gin.Context, err error) {
	statusCode := http.StatusBadRequest
	msg := "Error during getting profile"
	if database.IsRecordNotFoundError(err) {
		statusCode = http.StatusNotFound
		msg = "Profile not found"
	}

	c.JSON(statusCode, pkgCommon.ErrorResponse{
		Code:    statusCode,
		Message: msg,
		Error:   err.Error(),
	})
}
