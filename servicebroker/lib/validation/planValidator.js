'use strict'

require 

var logger = require('../log/logger');
var catalog = require('../../config/catalog.json')

var createErrorResponse = function(property,message,id) {
  var errorObject = { property:property,
  message: message, id:id }; 
  return errorObject;
}

var valueInPlan = function(plan_id, key){
    for (i = 0; i < catalog.services.length; i++){
        if(catalog.services[i].name == "autoscaler"){
            for ( j = 0; j < catalog.services[i].plans.length; j++){
                if(catalog.services[i].plans[j].id == plan_id){
                    return catalog.services[i].plans[j][key];
                }
            }
        }
    }
}

var validatePlanJSONValues = function(policyJson, plan_id) {
    var errors = [];
    var errorCount = 0;
    if(policyJson.schedules){
        if( (policyJson.schedules.recurring_schedule) && (policyJson.schedules.recurring_schedule.length > valueInPlan(plan_id, "recurring_schedule_count")) ){
            let error = createErrorResponse('schedules.recurring_schedule', 'policy exceeded recurring_schedule as per plan', plan_id)
            errors[errorCount++] = error
        }
        else{
            if( (policyJson.schedules.specific_date) && (policyJson.schedules.specific_date.length > valueInPlan(plan_id, "specific_date_count")) ){
                let error = createErrorResponse('schedules.specific_date', 'policy exceeded specific_date as per plan', plan_id)
                errors[errorCount++] = error
            }
        }
    }
    return errors
}

exports.validatePlan = function(req, callback) {
  var inputPolicy = req.body.parameters;
  var plan_id = req.body.plan_id;
  if(callback) {
    var errors = validatePlanJSONValues(inputPolicy, plan_id); 
    callback(errors);
  }
  else{
    logger.error('No callback function specified!', {});
    return;
  }
}