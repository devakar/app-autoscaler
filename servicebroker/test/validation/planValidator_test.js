'use strict';

var expect = require("chai").expect;
var fs = require('fs');
var path = require('path');
var planValidator = require(path.join(__dirname,'../../lib/validation/planValidator'));

describe('Validating Plan',function(){
    var fakePolicy;

    beforeEach(function(){
      fakePolicy = JSON.parse(fs.readFileSync(__dirname+'/../fakePolicy.json', 'utf8'));
    });
    
    it('Should validate the plan successfully when there is no scaling rule, recurring and specific date schedule',function(){
      delete fakePolicy.schedules.specific_date;
      delete fakePolicy.schedules.recurring_schedule;
      delete fakePolicy.scaling_rules;
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result).to.be.empty;
      });
    });

    it('Should validate the plan successfully when only recurring schedules exist',function(){
      delete fakePolicy.schedules.specific_date;
      delete fakePolicy.schedules.recurring_schedule;
      delete fakePolicy.scaling_rules;
      fakePolicy.schedules.recurring_schedule = [{
           "start_time":"10:00",
            "end_time":"18:00",
            "days_of_week":[
               1,
               2,
               3
            ],
            "instance_min_count":1,
            "instance_max_count":10,
            "initial_min_instance_count":5
      }]
      
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result).to.be.empty;
      });
    });

    it('Should validate the plan successfully when only specific date schedules exist',function(){
      delete fakePolicy.schedules.specific_date;
      delete fakePolicy.schedules.recurring_schedule;
      delete fakePolicy.scaling_rules;
      fakePolicy.schedules.specific_date = [{
           "start_date_time":"2015-06-02T10:00",
            "end_date_time":"2015-06-15T13:59",
            "instance_min_count":1,
            "instance_max_count":4,
            "initial_min_instance_count":2
      }]      
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result).to.be.empty;
      });
    });

    it('Should validate the plan successfully when only scaling rules exist',function(){
      delete fakePolicy.schedules.specific_date;
      delete fakePolicy.schedules.recurring_schedule;
      delete fakePolicy.scaling_rules;
      fakePolicy.scaling_rules = [{
         "metric_type":"memoryused",
         "stat_window_secs":300,
         "breach_duration_secs":600,
         "threshold":30,
         "operator":"<",
         "cool_down_secs":300,
         "adjustment":"-1"
      }]      
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result).to.be.empty;
      });
    });

    it('Should validate the plan successfully when recurring and specific date schedules exist',function(){
      delete fakePolicy.schedules.specific_date;
      delete fakePolicy.schedules.recurring_schedule;
      delete fakePolicy.scaling_rules;
      fakePolicy.schedules.specific_date = [{
           "start_date_time":"2015-06-02T10:00",
            "end_date_time":"2015-06-15T13:59",
            "instance_min_count":1,
            "instance_max_count":4,
            "initial_min_instance_count":2
      }]
      fakePolicy.schedules.recurring_schedule = [{
           "start_time":"10:00",
            "end_time":"18:00",
            "days_of_week":[
               1,
               2,
               3
            ],
            "instance_min_count":1,
            "instance_max_count":10,
            "initial_min_instance_count":5
      }]
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result).to.be.empty;
      });
    });

    it('Should validate the plan successfully when scaling rules and recurring schedules exist',function(){
      delete fakePolicy.schedules.specific_date;
      delete fakePolicy.schedules.recurring_schedule;
      delete fakePolicy.scaling_rules;
      fakePolicy.scaling_rules = [{
         "metric_type":"memoryused",
         "stat_window_secs":300,
         "breach_duration_secs":600,
         "threshold":30,
         "operator":"<",
         "cool_down_secs":300,
         "adjustment":"-1"
      }]
      fakePolicy.schedules.recurring_schedule = [{
           "start_time":"10:00",
            "end_time":"18:00",
            "days_of_week":[
               1,
               2,
               3
            ],
            "instance_min_count":1,
            "instance_max_count":10,
            "initial_min_instance_count":5
      }]
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result).to.be.empty;
      });
    });

    it('Should validate the plan successfully when scaling rules and specific date schedules exist',function(){
      delete fakePolicy.schedules.specific_date;
      delete fakePolicy.schedules.recurring_schedule;
      delete fakePolicy.scaling_rules;
      fakePolicy.scaling_rules = [{
         "metric_type":"memoryused",
         "stat_window_secs":300,
         "breach_duration_secs":600,
         "threshold":30,
         "operator":"<",
         "cool_down_secs":300,
         "adjustment":"-1"
      }]
      fakePolicy.schedules.specific_date = [{
           "start_date_time":"2015-06-02T10:00",
            "end_date_time":"2015-06-15T13:59",
            "instance_min_count":1,
            "instance_max_count":4,
            "initial_min_instance_count":2
      }]
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result).to.be.empty;
      });
    });

    it('Should validate the plan successfully when scaling rules, recurring and specific date schedules exist',function(){
      delete fakePolicy.schedules.specific_date;
      delete fakePolicy.schedules.recurring_schedule;
      delete fakePolicy.scaling_rules;
      fakePolicy.scaling_rules = [{
         "metric_type":"memoryused",
         "stat_window_secs":300,
         "breach_duration_secs":600,
         "threshold":30,
         "operator":"<",
         "cool_down_secs":300,
         "adjustment":"-1"
      }]
      fakePolicy.schedules.specific_date = [{
           "start_date_time":"2015-06-02T10:00",
            "end_date_time":"2015-06-15T13:59",
            "instance_min_count":1,
            "instance_max_count":4,
            "initial_min_instance_count":2
      }]
      fakePolicy.schedules.recurring_schedule = [{
           "start_time":"10:00",
            "end_time":"18:00",
            "days_of_week":[
               1,
               2,
               3
            ],
            "instance_min_count":1,
            "instance_max_count":10,
            "initial_min_instance_count":5
      }]
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result).to.be.empty;
      });
    });

    it('Should failed to validate the plan due to  exceeded maximum limit of recuuring schedules as per plan',function(){
      delete fakePolicy.schedules.specific_date;
      delete fakePolicy.scaling_rules;
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result[0]).to.have.property('message').and.equal('policy exceeded recurring_schedule as per plan of service');
        expect(result[0]).to.have.property('property').and.equal('schedules.recurring_schedule');
      });
    });

    it('Should failed to validate the plan due to  exceeded maximum limit of specific date schedules as per plan',function(){
      delete fakePolicy.schedules.recurring_schedule;
      delete fakePolicy.scaling_rules;
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result[0]).to.have.property('message').and.equal('policy exceeded specific_date as per plan of service');
        expect(result[0]).to.have.property('property').and.equal('schedules.specific_date');
      });
    });

    it('Should failed to validate the plan due to  exceeded maximum limit of scaling rules as per plan',function(){
      delete fakePolicy.schedules.specific_date;
      delete fakePolicy.schedules.recurring_schedule;
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result[0]).to.have.property('message').and.equal('policy exceeded scaling rules as per plan of service');
        expect(result[0]).to.have.property('property').and.equal('scaling_rules');
      });
    });
    
    it('Should failed to validate the plan due to  exceeded maximum limit of recurring schedules and specific date schedules as per plan',function(){
      delete fakePolicy.scaling_rules;
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result[0]).to.have.property('message').and.equal('policy exceeded recurring_schedule as per plan of service');
        expect(result[0]).to.have.property('property').and.equal('schedules.recurring_schedule');
        expect(result[1]).to.have.property('message').and.equal('policy exceeded specific_date as per plan of service');
        expect(result[1]).to.have.property('property').and.equal('schedules.specific_date');
      });
    });

    it('Should failed to validate the plan due to  exceeded maximum limit of scaling rules and  recurring schedules as per plan',function(){
      delete fakePolicy.schedules.specific_date;
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result[0]).to.have.property('message').and.equal('policy exceeded recurring_schedule as per plan of service');
        expect(result[0]).to.have.property('property').and.equal('schedules.recurring_schedule');
        expect(result[1]).to.have.property('message').and.equal('policy exceeded scaling rules as per plan of service');
        expect(result[1]).to.have.property('property').and.equal('scaling_rules');
      });
    });

    it('Should failed to validate the plan due to  exceeded maximum limit of scaling rules and specific date schedules as per plan',function(){
      delete fakePolicy.schedules.recurring_schedule;
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result[0]).to.have.property('message').and.equal('policy exceeded specific_date as per plan of service');
        expect(result[0]).to.have.property('property').and.equal('schedules.specific_date');
        expect(result[1]).to.have.property('message').and.equal('policy exceeded scaling rules as per plan of service');
        expect(result[1]).to.have.property('property').and.equal('scaling_rules');
      });
    });

    it('Should failed to validate the plan due to  exceeded maximum limit of scaling rules, recurring schedules and specific date schedules as per plan',function(){
      planValidator.validatePlan(fakePolicy,"autoscaler-guid", "autoscaler-free-plan-id", function(result){
        expect(result[0]).to.have.property('message').and.equal('policy exceeded recurring_schedule as per plan of service');
        expect(result[0]).to.have.property('property').and.equal('schedules.recurring_schedule');
        expect(result[1]).to.have.property('message').and.equal('policy exceeded specific_date as per plan of service');
        expect(result[1]).to.have.property('property').and.equal('schedules.specific_date');
        expect(result[2]).to.have.property('message').and.equal('policy exceeded scaling rules as per plan of service');
        expect(result[2]).to.have.property('property').and.equal('scaling_rules');
      });
    });
});