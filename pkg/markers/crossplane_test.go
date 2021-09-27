package markers

import (
	"testing"

	"github.com/crossplane-contrib/terrajet/pkg/terraform/resource"

	"github.com/google/go-cmp/cmp"
)

func TestCrossplaneOptions_String(t *testing.T) {
	type args struct {
		referenceToType            string
		referenceExtractor         string
		referenceFieldName         string
		referenceSelectorFieldName string
	}
	type want struct {
		out string
	}
	cases := map[string]struct {
		args
		want
	}{
		"NoOption": {
			args: args{
				referenceToType: "",
			},
			want: want{
				out: "",
			},
		},
		"WithType": {
			args: args{
				referenceToType: "SecurityGroup",
			},
			want: want{
				out: "+crossplane:generate:reference:type=SecurityGroup\n",
			},
		},
		"WithAll": {
			args: args{
				referenceToType:            "github.com/crossplane/provider-aws/apis/ec2/v1beta1.Subnet",
				referenceExtractor:         "github.com/crossplane/provider-aws/apis/ec2/v1beta1.SubnetARN()",
				referenceFieldName:         "SubnetIDRefs",
				referenceSelectorFieldName: "SubnetIDSelector",
			},
			want: want{
				out: `+crossplane:generate:reference:type=github.com/crossplane/provider-aws/apis/ec2/v1beta1.Subnet
+crossplane:generate:reference:extractor=github.com/crossplane/provider-aws/apis/ec2/v1beta1.SubnetARN()
+crossplane:generate:reference:refFieldName=SubnetIDRefs
+crossplane:generate:reference:selectorFieldName=SubnetIDSelector
`,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			o := CrossplaneOptions{
				FieldReferenceConfiguration: resource.FieldReferenceConfiguration{
					ReferenceToType:            tc.referenceToType,
					ReferenceExtractor:         tc.referenceExtractor,
					ReferenceFieldName:         tc.referenceFieldName,
					ReferenceSelectorFieldName: tc.referenceSelectorFieldName,
				},
			}
			got := o.String()
			if diff := cmp.Diff(tc.want.out, got); diff != "" {
				t.Errorf("CrossplaneOptions.String(): -want result, +got result: %s", diff)
			}
		})
	}
}
