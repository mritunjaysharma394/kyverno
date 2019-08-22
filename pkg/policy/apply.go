package policy

import (
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/glog"
	kyverno "github.com/nirmata/kyverno/pkg/api/kyverno/v1alpha1"
	"github.com/nirmata/kyverno/pkg/engine"
	"github.com/nirmata/kyverno/pkg/info"
	"github.com/nirmata/kyverno/pkg/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// applyPolicy applies policy on a resource
//TODO: generation rules
func applyPolicy(policy kyverno.Policy, resource unstructured.Unstructured, policyStatus PolicyStatusInterface) (info.PolicyInfo, error) {
	var ps PolicyStat
	gatherStat := func(policyName string, er engine.EngineResponse) {
		// ps := policyctr.PolicyStat{}
		ps.PolicyName = policyName
		ps.Stats.ValidationExecutionTime = er.ExecutionTime
		ps.Stats.RulesAppliedCount = er.RulesAppliedCount
	}
	// send stats for aggregation
	sendStat := func(blocked bool) {
		//SEND
		policyStatus.SendStat(ps)
	}

	startTime := time.Now()
	glog.V(4).Infof("Started apply policy %s on resource %s/%s/%s (%v)", policy.Name, resource.GetKind(), resource.GetNamespace(), resource.GetName(), startTime)
	defer func() {
		glog.V(4).Infof("Finished applying %s on resource %s/%s/%s (%v)", policy.Name, resource.GetKind(), resource.GetNamespace(), resource.GetName(), time.Since(startTime))
	}()
	// glog.V(4).Infof("apply policy %s with resource version %s on resource %s/%s/%s with resource version %s", policy.Name, policy.ResourceVersion, resource.GetKind(), resource.GetNamespace(), resource.GetName(), resource.GetResourceVersion())
	policyInfo := info.NewPolicyInfo(policy.Name, resource.GetKind(), resource.GetName(), resource.GetNamespace(), policy.Spec.ValidationFailureAction)

	//MUTATION
	mruleInfos, err := mutation(policy, resource, policyStatus)
	policyInfo.AddRuleInfos(mruleInfos)
	if err != nil {
		return policyInfo, err
	}

	//VALIDATION
	engineResponse := engine.Validate(policy, resource)
	if len(engineResponse.RuleInfos) != 0 {
		policyInfo.AddRuleInfos(engineResponse.RuleInfos)
	}
	// gather stats
	gatherStat(policy.Name, engineResponse)
	//send stats
	sendStat(false)

	//TODO: GENERATION
	return policyInfo, nil
}

func mutation(policy kyverno.Policy, resource unstructured.Unstructured, policyStatus PolicyStatusInterface) ([]info.RuleInfo, error) {
	var ps PolicyStat
	// gather stats from the engine response
	gatherStat := func(policyName string, er engine.EngineResponse) {
		// ps := policyctr.PolicyStat{}
		ps.PolicyName = policyName
		ps.Stats.MutationExecutionTime = er.ExecutionTime
		ps.Stats.RulesAppliedCount = er.RulesAppliedCount
	}
	// send stats for aggregation
	sendStat := func(blocked bool) {
		//SEND
		policyStatus.SendStat(ps)
	}

	engineResponse := engine.Mutate(policy, resource)
	// gather stats
	gatherStat(policy.Name, engineResponse)
	//send stats
	sendStat(false)

	patches := engineResponse.Patches
	ruleInfos := engineResponse.RuleInfos
	if len(ruleInfos) == 0 {
		//no rules processed
		return nil, nil
	}

	for _, r := range ruleInfos {
		if !r.IsSuccessful() {
			// no failures while processing rule
			return ruleInfos, nil
		}
	}
	if len(patches) == 0 {
		// no patches for the resources
		// either there were failures or the overlay already was satisfied
		return ruleInfos, nil
	}

	// (original resource + patch) == (original resource)
	mergePatches := utils.JoinPatches(patches)
	patch, err := jsonpatch.DecodePatch(mergePatches)
	if err != nil {
		return nil, err
	}
	rawResource, err := resource.MarshalJSON()
	if err != nil {
		glog.V(4).Infof("unable to marshal resource : %v", err)
		return nil, err
	}

	// apply the patches returned by mutate to the original resource
	patchedResource, err := patch.Apply(rawResource)
	if err != nil {
		return nil, err
	}
	//TODO: this will be removed after the support for patching for each rule
	ruleInfo := info.NewRuleInfo("over-all mutation", info.Mutation)

	if !jsonpatch.Equal(patchedResource, rawResource) {
		//resource does not match so there was a mutation rule violated
		// TODO : check the rule name "mutation rules"
		ruleInfo.Fail()
		ruleInfo.Add("resource does not satisfy mutation rules")
	} else {
		ruleInfo.Add("resource satisfys the mutation rule")
	}

	ruleInfos = append(ruleInfos, ruleInfo)
	return ruleInfos, nil
}