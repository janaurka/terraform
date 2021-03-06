package cfapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/cf/api/organizations"
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/net"
)

// OrgManager -
type OrgManager struct {
	config    coreconfig.Reader
	ccGateway net.Gateway

	apiEndpoint string

	repo organizations.OrganizationRepository
}

// CCOrg -
type CCOrg struct {
	ID string

	Name      string `json:"name"`
	Status    string `json:"status,omitempty"`
	QuotaGUID string `json:"quota_definition_guid,omitempty"`
}

// CCOrgResource -
type CCOrgResource struct {
	Metadata resources.Metadata `json:"metadata"`
	Entity   CCOrg              `json:"entity"`
}

// OrgRole -
type OrgRole string

// OrgRoleMember -
const OrgRoleMember = OrgRole("users")

// OrgRoleManager -
const OrgRoleManager = OrgRole("managers")

// OrgRoleBillingManager -
const OrgRoleBillingManager = OrgRole("billing_managers")

// OrgRoleAuditor -
const OrgRoleAuditor = OrgRole("auditors")

// NewOrgManager -
func NewOrgManager(config coreconfig.Reader, ccGateway net.Gateway) (dm *OrgManager, err error) {

	dm = &OrgManager{
		config:    config,
		ccGateway: ccGateway,

		apiEndpoint: config.APIEndpoint(),

		repo: organizations.NewCloudControllerOrganizationRepository(config, ccGateway),
	}

	if dm.apiEndpoint == "" {
		err = errors.New("API endpoint missing from config file")
		return
	}

	return
}

// FindOrg -
func (om *OrgManager) FindOrg(name string) (org CCOrg, err error) {
	orgModel, err := om.repo.FindByName(name)
	org.ID = orgModel.GUID
	org.Name = orgModel.Name
	return
}

// ReadOrg -
func (om *OrgManager) ReadOrg(orgID string) (org CCOrg, err error) {

	resource := &CCOrgResource{}
	err = om.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/organizations/%s", om.apiEndpoint, orgID), &resource)

	org = resource.Entity
	org.ID = resource.Metadata.GUID
	return
}

// CreateOrg -
func (om *OrgManager) CreateOrg(name string, quotaID string) (org CCOrg, err error) {

	payload := map[string]interface{}{"name": name}
	if len(quotaID) > 0 {
		payload["quota_definition_guid"] = quotaID
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	resource := CCOrgResource{}
	if err = om.ccGateway.CreateResource(om.apiEndpoint,
		"/v2/organizations", bytes.NewReader(body), &resource); err != nil {
		return
	}
	org = resource.Entity
	org.ID = resource.Metadata.GUID
	return
}

// UpdateOrg -
func (om *OrgManager) UpdateOrg(org CCOrg) (err error) {

	body, err := json.Marshal(org)

	request, err := om.ccGateway.NewRequest("PUT",
		fmt.Sprintf("%s/v2/organizations/%s", om.apiEndpoint, org.ID),
		om.config.AccessToken(), bytes.NewReader(body))
	if err != nil {
		return err
	}

	resource := &CCOrgResource{}
	_, err = om.ccGateway.PerformRequestForJSONResponse(request, resource)
	return
}

// AddUsers -
func (om *OrgManager) AddUsers(orgID string, userIDs []string, role OrgRole) (err error) {

	for _, uid := range userIDs {
		err = om.ccGateway.UpdateResource(om.apiEndpoint,
			fmt.Sprintf("/v2/organizations/%s/%s/%s", orgID, role, uid),
			strings.NewReader(""))
	}
	return
}

// RemoveUsers -
func (om *OrgManager) RemoveUsers(orgID string, userIDs []string, role OrgRole) (err error) {

	for _, uid := range userIDs {
		err = om.ccGateway.DeleteResource(om.apiEndpoint,
			fmt.Sprintf("/v2/organizations/%s/%s/%s", orgID, role, uid))
	}
	return
}

// ListUsers -
func (om *OrgManager) ListUsers(orgID string, role OrgRole) (userIDs []interface{}, err error) {

	userList := &CCUserList{}
	err = om.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/organizations/%s/%s", om.apiEndpoint, orgID, role), userList)
	for _, r := range userList.Resources {
		userIDs = append(userIDs, r.Metadata.GUID)
	}
	return
}

// DeleteOrg -
func (om *OrgManager) DeleteOrg(id string) (err error) {
	err = om.ccGateway.DeleteResource(om.apiEndpoint, fmt.Sprintf("/v2/organizations/%s", id))
	return
}
