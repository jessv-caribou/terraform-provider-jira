package jira

import (
	"fmt"

	jira "github.com/andygrunwald/go-jira"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/pkg/errors"
)

type userSearchRequest struct {
	query string `json:"query,omitempty" structs:"query,omitempty"`
}

type jiraCloudUser struct {
	Self         string          `json:"self,omitempty" structs:"self,omitempty"`
	AccountID    string          `json:"accountId,omitempty" structs:"accountId,omitempty"`
	AccountType  string          `json:"accountType,omitempty" structs:"accountType,omitempty"`
	EmailAddress string          `json:"emailAddress,omitempty" structs:"emailAddress,omitempty"`
	AvatarUrls   jira.AvatarUrls `json:"avatarUrls,omitempty" structs:"avatarUrls,omitempty"`
	DisplayName  string          `json:"displayName,omitempty" structs:"displayName,omitempty"`
	Active       bool            `json:"active,omitempty" structs:"active,omitempty"`
	TimeZone     string          `json:"timeZone,omitempty" structs:"timeZone,omitempty"`
	Locale       string          `json:"locale,omitempty" structs:"locale,omitempty"`
}

func usernameFallbackSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	if new == "" {
		return old == d.Get("name")
	}
	return old == new
}

// resourceUser is used to define a JIRA issue
func resourceUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceUserCreate,
		Read:   resourceUserRead,
		Delete: resourceUserDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"email": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"display_name": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				DiffSuppressFunc: usernameFallbackSuppressFunc,
			},
		},
	}
}

func dataUser() *schema.Resource {
	return &schema.Resource{
		Read: dataUserRead,
		Schema: map[string]*schema.Schema{
			"account_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"email": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"display_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func getUserByKey(client *jira.Client, key string) (*jira.User, *jira.Response, error) {
	apiEndpoint := fmt.Sprintf("%s?key=%s", userAPIEndpoint, key)
	req, err := client.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}

	user := new(jira.User)
	resp, err := client.Do(req, user)
	if err != nil {
		return nil, resp, jira.NewJiraError(resp, err)
	}
	return user, resp, nil
}

func getUserByUsername(client *jira.Client, email string) (*jiraCloudUser, *jira.Response, error) {
	apiEndpoint := "rest/api/3/user/search?query=" + email
	fmt.Println("API Endpoint: " + apiEndpoint)
	req, err := client.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}

	users := new([]jiraCloudUser)

	resp, err := client.Do(req, users)

	var userIndex int = -1

	for i, user := range *users {
		if user.AccountType == "atlassian" {
			userIndex = i
			break
		}
	}

	if userIndex < 0 {
		err := errors.New("jiraClient: User does not exist")
		return nil, resp, err
	}

	user := (*users)[userIndex]

	if err != nil {
		return nil, resp, jira.NewJiraError(resp, err)
	}
	return &user, resp, nil
}

func deleteUserByKey(client *jira.Client, key string) (*jira.Response, error) {
	apiEndpoint := fmt.Sprintf("%s?key=%s", userAPIEndpoint, key)
	req, err := client.NewRequest("DELETE", apiEndpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req, nil)
	if err != nil {
		return resp, jira.NewJiraError(resp, err)
	}
	return resp, nil
}

// resourceUserCreate creates a new jira user using the jira api
func resourceUserCreate(d *schema.ResourceData, m interface{}) error {
	config := m.(*Config)

	user := new(jira.User)
	user.Name = d.Get("name").(string)
	user.EmailAddress = d.Get("email").(string)

	dn, ok := d.GetOkExists("display_name")
	user.DisplayName = dn.(string)

	if !ok {
		user.DisplayName = user.Name
	}

	createdUser, _, err := config.jiraClient.User.Create(user)

	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	d.SetId(createdUser.Key)

	return resourceUserRead(d, m)
}

func dataUserRead(d *schema.ResourceData, m interface{}) error {
	config := m.(*Config)

	user, _, err := getUserByUsername(config.jiraClient, d.Get("email").(string))
	if err != nil {
		return errors.Wrap(err, "getting jira user failed")
	}
	d.SetId(user.AccountID)
	d.Set("email", user.EmailAddress)
	d.Set("account_id", user.AccountID)
	d.Set("display_name", user.DisplayName)
	return nil
}

// resourceUserRead reads issue details using jira api
func resourceUserRead(d *schema.ResourceData, m interface{}) error {
	config := m.(*Config)

	user, _, err := getUserByKey(config.jiraClient, d.Id())
	if err != nil {
		return errors.Wrap(err, "getting jira user failed")
	}

	d.Set("name", user.Name)
	d.Set("display_name", user.DisplayName)
	d.Set("email", user.EmailAddress)
	return nil
}

// resourceUserDelete deletes jira issue using the jira api
func resourceUserDelete(d *schema.ResourceData, m interface{}) error {
	config := m.(*Config)

	_, err := deleteUserByKey(config.jiraClient, d.Id())

	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	return nil
}
