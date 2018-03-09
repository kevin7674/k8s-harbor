// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dao

import (
	"fmt"
	"time"

	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
)

// AddRepTarget ...
func AddRepTarget(target models.RepTarget) (int64, error) {
	o := GetOrmer()
	return o.Insert(&target)
}

// GetRepTarget ...
func GetRepTarget(id int64) (*models.RepTarget, error) {
	o := GetOrmer()
	t := models.RepTarget{ID: id}
	err := o.Read(&t)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// GetRepTargetByName ...
func GetRepTargetByName(name string) (*models.RepTarget, error) {
	o := GetOrmer()
	t := models.RepTarget{Name: name}
	err := o.Read(&t, "Name")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// GetRepTargetByEndpoint ...
func GetRepTargetByEndpoint(endpoint string) (*models.RepTarget, error) {
	o := GetOrmer()
	t := models.RepTarget{
		URL: endpoint,
	}
	err := o.Read(&t, "URL")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// DeleteRepTarget ...
func DeleteRepTarget(id int64) error {
	o := GetOrmer()
	_, err := o.Delete(&models.RepTarget{ID: id})
	return err
}

// UpdateRepTarget ...
func UpdateRepTarget(target models.RepTarget) error {
	o := GetOrmer()
	target.UpdateTime = time.Now()
	_, err := o.Update(&target, "URL", "Name", "Username", "Password", "Insecure", "UpdateTime")
	return err
}

// FilterRepTargets filters targets by name
func FilterRepTargets(name string) ([]*models.RepTarget, error) {
	o := GetOrmer()

	var args []interface{}

	sql := `select * from replication_target `
	if len(name) != 0 {
		sql += `where name like ? `
		args = append(args, "%"+escape(name)+"%")
	}
	sql += `order by creation_time`

	var targets []*models.RepTarget

	if _, err := o.Raw(sql, args).QueryRows(&targets); err != nil {
		return nil, err
	}

	return targets, nil
}

// AddRepPolicy ...
func AddRepPolicy(policy models.RepPolicy) (int64, error) {
	o := GetOrmer()
	sql := `insert into replication_policy (name, project_id, target_id, enabled, description, cron_str, creation_time, update_time, filters, replicate_deletion) 
				values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	params := []interface{}{}
	now := time.Now()
	params = append(params, policy.Name, policy.ProjectID, policy.TargetID, 1,
		policy.Description, policy.Trigger, now, now, policy.Filters,
		policy.ReplicateDeletion)

	result, err := o.Raw(sql, params...).Exec()
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetRepPolicy ...
func GetRepPolicy(id int64) (*models.RepPolicy, error) {
	o := GetOrmer()
	sql := `select * from replication_policy where id = ? and deleted = 0`

	var policy models.RepPolicy

	if err := o.Raw(sql, id).QueryRow(&policy); err != nil {
		if err == orm.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

// GetTotalOfRepPolicies returns the total count of replication policies
func GetTotalOfRepPolicies(name string, projectID int64) (int64, error) {
	qs := GetOrmer().QueryTable(&models.RepPolicy{}).Filter("deleted", 0)

	if len(name) != 0 {
		qs = qs.Filter("name__icontains", name)
	}

	if projectID != 0 {
		qs = qs.Filter("project_id", projectID)
	}

	return qs.Count()
}

// FilterRepPolicies filters policies by name and project ID
func FilterRepPolicies(name string, projectID, page, pageSize int64) ([]*models.RepPolicy, error) {
	o := GetOrmer()

	var args []interface{}

	sql := `select rp.id, rp.project_id, rp.target_id, 
				rt.name as target_name, rp.name, rp.description,
				rp.cron_str, rp.filters, rp.replicate_deletion, 
				rp.creation_time, rp.update_time, 
				count(rj.status) as error_job_count 
			from replication_policy rp 
			left join replication_target rt on rp.target_id=rt.id 
			left join replication_job rj on rp.id=rj.policy_id and (rj.status="error" 
				or rj.status="retrying") 
			where rp.deleted = 0 `

	if len(name) != 0 && projectID != 0 {
		sql += `and rp.name like ? and rp.project_id = ? `
		args = append(args, "%"+escape(name)+"%")
		args = append(args, projectID)
	} else if len(name) != 0 {
		sql += `and rp.name like ? `
		args = append(args, "%"+escape(name)+"%")
	} else if projectID != 0 {
		sql += `and rp.project_id = ? `
		args = append(args, projectID)
	}

	sql += `group by rp.id order by rp.creation_time`

	if page > 0 && pageSize > 0 {
		sql += ` limit ? offset ?`
		args = append(args, pageSize, (page-1)*pageSize)
	}

	var policies []*models.RepPolicy
	if _, err := o.Raw(sql, args).QueryRows(&policies); err != nil {
		return nil, err
	}

	return policies, nil
}

// GetRepPolicyByName ...
func GetRepPolicyByName(name string) (*models.RepPolicy, error) {
	o := GetOrmer()
	sql := `select * from replication_policy where deleted = 0 and name = ?`

	var policy models.RepPolicy

	if err := o.Raw(sql, name).QueryRow(&policy); err != nil {
		if err == orm.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &policy, nil
}

// GetRepPolicyByProject ...
func GetRepPolicyByProject(projectID int64) ([]*models.RepPolicy, error) {
	o := GetOrmer()
	sql := `select * from replication_policy where deleted = 0 and project_id = ?`

	var policies []*models.RepPolicy

	if _, err := o.Raw(sql, projectID).QueryRows(&policies); err != nil {
		return nil, err
	}

	return policies, nil
}

// GetRepPolicyByTarget ...
func GetRepPolicyByTarget(targetID int64) ([]*models.RepPolicy, error) {
	o := GetOrmer()
	sql := `select * from replication_policy where deleted = 0 and target_id = ?`

	var policies []*models.RepPolicy

	if _, err := o.Raw(sql, targetID).QueryRows(&policies); err != nil {
		return nil, err
	}

	return policies, nil
}

// GetRepPolicyByProjectAndTarget ...
func GetRepPolicyByProjectAndTarget(projectID, targetID int64) ([]*models.RepPolicy, error) {
	o := GetOrmer()
	sql := `select * from replication_policy where deleted = 0 and project_id = ? and target_id = ?`

	var policies []*models.RepPolicy

	if _, err := o.Raw(sql, projectID, targetID).QueryRows(&policies); err != nil {
		return nil, err
	}

	return policies, nil
}

// UpdateRepPolicy ...
func UpdateRepPolicy(policy *models.RepPolicy) error {
	o := GetOrmer()
	policy.UpdateTime = time.Now()
	_, err := o.Update(policy, "ProjectID", "TargetID", "Name", "Description",
		"Trigger", "Filters", "ReplicateDeletion", "UpdateTime")
	return err
}

// DeleteRepPolicy ...
func DeleteRepPolicy(id int64) error {
	o := GetOrmer()
	policy := &models.RepPolicy{
		ID:         id,
		Deleted:    1,
		UpdateTime: time.Now(),
	}
	_, err := o.Update(policy, "Deleted")
	return err
}

// AddRepJob ...
func AddRepJob(job models.RepJob) (int64, error) {
	o := GetOrmer()
	if len(job.Status) == 0 {
		job.Status = models.JobPending
	}
	if len(job.TagList) > 0 {
		job.Tags = strings.Join(job.TagList, ",")
	}
	return o.Insert(&job)
}

// GetRepJob ...
func GetRepJob(id int64) (*models.RepJob, error) {
	o := GetOrmer()
	j := models.RepJob{ID: id}
	err := o.Read(&j)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	genTagListForJob(&j)
	return &j, nil
}

// GetRepJobByPolicy ...
func GetRepJobByPolicy(policyID int64) ([]*models.RepJob, error) {
	var res []*models.RepJob
	_, err := repJobPolicyIDQs(policyID).All(&res)
	genTagListForJob(res...)
	return res, err
}

// FilterRepJobs ...
func FilterRepJobs(policyID int64, repository string, status []string, startTime,
	endTime *time.Time, limit, offset int64) ([]*models.RepJob, int64, error) {

	jobs := []*models.RepJob{}

	qs := GetOrmer().QueryTable(new(models.RepJob))

	if policyID != 0 {
		qs = qs.Filter("PolicyID", policyID)
	}
	if len(repository) != 0 {
		qs = qs.Filter("Repository__icontains", repository)
	}
	if len(status) != 0 {
		qs = qs.Filter("Status__in", status)
	}
	if startTime != nil {
		qs = qs.Filter("CreationTime__gte", startTime)
	}
	if endTime != nil {
		qs = qs.Filter("CreationTime__lte", endTime)
	}

	total, err := qs.Count()
	if err != nil {
		return jobs, 0, err
	}

	qs = qs.OrderBy("-UpdateTime")

	_, err = qs.Limit(limit).Offset(offset).All(&jobs)
	if err != nil {
		return jobs, 0, err
	}

	genTagListForJob(jobs...)

	return jobs, total, nil
}

// GetRepJobToStop get jobs that are possibly being handled by workers of a certain policy.
func GetRepJobToStop(policyID int64) ([]*models.RepJob, error) {
	var res []*models.RepJob
	_, err := repJobPolicyIDQs(policyID).Filter("status__in",
		models.JobPending, models.JobRunning, models.JobRetrying).All(&res)
	genTagListForJob(res...)
	return res, err
}

func repJobQs() orm.QuerySeter {
	o := GetOrmer()
	return o.QueryTable("replication_job")
}

func repJobPolicyIDQs(policyID int64) orm.QuerySeter {
	return repJobQs().Filter("policy_id", policyID)
}

// DeleteRepJob ...
func DeleteRepJob(id int64) error {
	o := GetOrmer()
	_, err := o.Delete(&models.RepJob{ID: id})
	return err
}

// UpdateRepJobStatus ...
func UpdateRepJobStatus(id int64, status string) error {
	o := GetOrmer()
	j := models.RepJob{
		ID:         id,
		Status:     status,
		UpdateTime: time.Now(),
	}
	n, err := o.Update(&j, "Status", "UpdateTime")
	if n == 0 {
		log.Warningf("no records are updated when updating replication job %d", id)
	}
	return err
}

// ResetRunningJobs update all running jobs status to pending, including replication jobs and scan jobs.
func ResetRunningJobs() error {
	o := GetOrmer()
	sql := fmt.Sprintf("update replication_job set status = '%s', update_time = ? where status = '%s'", models.JobPending, models.JobRunning)
	_, err := o.Raw(sql, time.Now()).Exec()
	if err != nil {
		return err
	}
	sql = fmt.Sprintf("update %s set status = '%s', update_time = ? where status = '%s'", models.ScanJobTable, models.JobPending, models.JobRunning)
	_, err = o.Raw(sql, time.Now()).Exec()
	return err
}

// GetRepJobByStatus get jobs of certain statuses
func GetRepJobByStatus(status ...string) ([]*models.RepJob, error) {
	var res []*models.RepJob
	var t []interface{}
	for _, s := range status {
		t = append(t, interface{}(s))
	}
	_, err := repJobQs().Filter("status__in", t...).All(&res)
	genTagListForJob(res...)
	return res, err
}

func genTagListForJob(jobs ...*models.RepJob) {
	for _, j := range jobs {
		if len(j.Tags) > 0 {
			j.TagList = strings.Split(j.Tags, ",")
		}
	}
}