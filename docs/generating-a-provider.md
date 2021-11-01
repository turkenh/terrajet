# Generating a Crossplane Provider from a Terraform Provider

In this Guide, we will generate a Crossplane provider leveraging an existing
Terraform provider using Terrajet. 

We have chosen [Terraform GitHub provider] as an example, but the process would
be identical for any other Terraform provider.

## Generate

1. Generate a GitHub repository for the Crossplane provider by hitting the
   "Use this template" button in [provider-tf-template] repository.
2. Clone the repository to your local.
3. Replace `template` with your provider name.
    1. Export `ProviderName`

    ```shell
    export ProviderNameLower=github
    export ProviderNameUpper=GitHub
    ```

    2. Replace all occurrences of `template` with your provider name.

    ```shell
    git grep -l 'template' -- './*' ':!build/**' ':!go.sum' | xargs sed -i.bak "s/template/${ProviderNameLower}/g"
    git grep -l 'Template' -- './*' ':!build/**' ':!go.sum' | xargs sed -i.bak "s/Template/${ProviderNameUpper}/g"
    # Clean up the .bak files created by sed
    git clean -fd
   
    mv "internal/clients/template.go" "internal/clients/${ProviderNameLower}.go"
    mv "cluster/images/provider-tf-template" "cluster/images/provider-tf-${ProviderNameLower}"
    mv "cluster/images/provider-tf-template-controller" "cluster/images/provider-tf-${ProviderNameLower}-controller"
    ```

    3. Commit the changes

   ```shell
   git add .
   git commit -s -S -m "Rename template to ${ProviderNameLower}"
   ```

4. Configure your repo with Terraform provider and schema:
   1. Update Makefile variables for Terraform Provider (`TERRAFORM_PROVIDER_*`)

       ```makefile
       export TERRAFORM_PROVIDER_SOURCE := integrations/github
       export TERRAFORM_PROVIDER_VERSION := 4.17.0
       export TERRAFORM_PROVIDER_DOWNLOAD_NAME := terraform-provider-github
       export TERRAFORM_PROVIDER_DOWNLOAD_URL_PREFIX := https://releases.hashicorp.com/terraform-provider-github/4.17.0
       ```
   2. Find go repository of the Terraform provider set import path for the
      package with function `func Provider() terraform.ResourceProvider`.

      ```diff
      --- a/config/provider.go
      +++ b/config/provider.go
      @@ -2,7 +2,7 @@ package config
 
      import (
              tjconfig "github.com/crossplane-contrib/terrajet/pkg/config"
      -       tf "github.com/hashicorp/terraform-provider-hashicups/hashicups"
      +       tf "github.com/turkenh/terraform-provider-github/v4/github"
      )
 
      const resourcePrefix = "github"
      ```

      Run:
      ```
      go mod tidy
      ```

   3. If your provider uses an old version (<v2) of
      `github.com/hashicorp/terraform-plugin-sdk`, initialize a Terrajet provider
      configuration as follows:

      ```go
      pc := tjconfig.NewProvider(conversion.GetV2ResourceMap(tf.Provider()), resourcePrefix, modulePath,
            tjconfig.WithDefaultResourceFn(defaultResourceFn))
      ```
      And in `go.mod` file, set replace as follows:
      ```
      replace github.com/hashicorp/terraform-plugin-sdk => ../../other-repos/terraform-plugin-sdk
      ```

      Run:
      ```
      go mod tidy
      ```

5. Implement `ProviderConfig` logic. In `provider-tf-template`, there is already
a boilerplate code in file `internal/clients/${ProviderNameLower}.go` which
takes care of properly fetching secret data referenced from `ProviderConfig`
resource.

For our GitHub provider, we need to check [Terraform documentation for provider
configuration] and provide the keys there:

```go
   const (
     keyBaseURL = "base_url"
     keyOwner = "owner"
     keyToken = "token"

     // GitHub credentials environment variable names
     envToken = "GITHUB_TOKEN"
   )

func TerraformSetupBuilder(version, providerSource, providerVersion string) terraform.SetupFn {
  ...
  // set provider configuration
  ps.Configuration = map[string]interface{}{}
  if v, ok := githubCreds[keyBaseURL]; ok {
      ps.Configuration[keyBaseURL] = v
  }
  if v, ok := githubCreds[keyOwner]; ok {
      ps.Configuration[keyOwner] = v
  }
  // set environment variables for sensitive provider configuration
  ps.Env = []string{
      fmt.Sprintf(fmtEnvVar, envToken, githubCreds[keyToken]),
  }
  ...
```

6. Before generating all resource that the provider has, let's go step by step
and only start with generating `github_repository` and `github_branch`
resources. 
To limit the resources to be generated, we need to provide an include list
option with `tjconfig.WithIncludeList` in file `config/provider.go`:

```go
 pc := tjconfig.NewProvider(tjconfig.GetV2ResourceMap(tf.Provider()), resourcePrefix, "github.com/crossplane-contrib/provider-tf-github",
     tjconfig.WithIncludeList([]string{
         "github_repository$",
         "github_branch$",
     }))
```

7. Finally, we would need to add some custom configurations for these two
resources as follows:

```shell
# Create custom configuration directory for whole repository group
mkdir config/repository
# Create custom configuration directory for whole branch group
mkdir config/branch
```

```shell
cat <<EOF > config/repository/config.go
package repository

import "github.com/crossplane-contrib/terrajet/pkg/config"

func Customize(p *config.Provider) {
	p.AddResourceConfigurator("github_repository", func(r *config.Resource) {
		r.Group = "repository"
	})
}
EOF
```

```shell
cat <<EOF > config/branch/config.go
package branch

import "github.com/crossplane-contrib/terrajet/pkg/config"

func Customize(p *config.Provider) {
	p.AddResourceConfigurator("github_branch", func(r *config.Resource) {
		r.Group = "branch"
		r.ExternalName = config.IdentifierFromProvider
		r.References["repository"] = config.Reference{
			Type: "github.com/crossplane-contrib/provider-tf-github/apis/repository/v1alpha1.Repository",
		}
	})
}
EOF
```

And register custom configurations in `config/provider.go`:

```diff
        tf "github.com/turkenh/terraform-provider-github/v4/github"
+       "github.com/crossplane-contrib/provider-tf-github/config/branch"
+       "github.com/crossplane-contrib/provider-tf-github/config/repository"
 )

 func GetProvider() *tjconfig.Provider {
        ...
        for _, configure := range []func(provider *tjconfig.Provider){
            add custom config functions
+           repository.Customize,
+           branch.Customize,
        } {
		    configure(pc)
        }
```

8. Now we can generate our Terrajet Provider:

```shell
make generate
```

## Test

Now let's test our generated resources.

1. First, we will create example resources under `examples` directory:

```shell
rm -rf examples/sample
mkdir examples/repository
mkdir examples/branch
```

```shell
cat <<EOF > examples/providerconfig/secret.yaml.tmpl
apiVersion: v1
kind: Secret
metadata:
  name: example-creds
  namespace: crossplane-system
type: Opaque
stringData:
  credentials: |
    {
      "token": "y0ur-t0k3n"
    }
EOF
```

```shell
cat <<EOF > examples/repository/repository.yaml
apiVersion: repository.github.tf.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: hello-crossplane
spec:
  forProvider:
    description: "Crossplane Github Provider generated with Terrajet"
    visibility: public
    template:
      - owner: crossplane-contrib
        repository: provider-tf-template
  providerConfigRef:
    name: default
EOF
```

```shell
cat <<EOF > examples/branch/branch.yaml
apiVersion: branch.github.tf.crossplane.io/v1alpha1
kind: Branch
metadata:
  name: hello-terrajet
spec:
  forProvider:
    branch: hello-terrajet
    repositoryRef:
      name: hello-crossplane
  providerConfigRef:
    name: default
EOF
```

2. Generate a [Personal Access Token](https://github.com/settings/tokens) for
your Github account with `repo/public_repo` and `delete_repo` scopes.

3. Create `examples/providerconfig/secret.yaml` from
`examples/providerconfig/secret.yaml.tmpl` and set your token in the file:

```shell
GITHUB_TOKEN=<your-token-here>
cat examples/providerconfig/secret.yaml.tmpl | sed -e "s/y0ur-t0k3n/${GITHUB_TOKEN}/g" > examples/providerconfig/secret.yaml
```

4. Apply CRDs:

```
kubectl apply -f package/crds
```

5. Apply ProviderConfig and example manifests:

```
kubectl apply -f examples/providerconfig/
kubectl apply -f examples/repository/repository.yaml
kubectl apply -f examples/branch/branch.yaml
```

6. Run the provider:

```
make run
```

7. Observe managed resources and wait until they are ready:

```
watch kubectl get managed
```

```
NAME                                                   READY   SYNCED   EXTERNAL-NAME                     AGE
branch.branch.github.tf.crossplane.io/hello-terrajet   True    True     hello-crossplane:hello-terrajet   9m44s

NAME                                                             READY   SYNCED   EXTERNAL-NAME      AGE
repository.repository.github.tf.crossplane.io/hello-crossplane   True    True     hello-crossplane   9m48s
```

8. Verify that repo `hello-crossplane` and branch `hello-terrajet` created under
your github account.

[Terraform GitHub provider]: https://registry.terraform.io/providers/integrations/github/latest/docs
[provider-tf-template]: https://github.com/crossplane-contrib/provider-tf-template
[Terraform documentation for provider configuration]: https://registry.terraform.io/providers/integrations/github/latest/docs#argument-reference