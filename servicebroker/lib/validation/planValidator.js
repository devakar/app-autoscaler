'use strict'
var path = require('path');
var logger = require(path.join(__dirname,'../../lib/logger/logger'));
var catalog = require(path.join(__dirname,'../../config/catalog.json'));

var createErrorResponse = function(property, message, plan_id, service_id) {
  var errorObject = { property:property,
  message: message, plan_id:plan_id, service_id:service_id }; 
  return errorObject;
}

var valueInPlan = function(service_id, plan_id, key){
    for (var i = 0; i < catalog.services.length; i++){
        if(catalog.services[i].id == service_id){
            for (var j = 0; j < catalog.services[i].plans.length; j++){
                if(catalog.services[i].plans[j].id == plan_id){
                    return catalog.services[i].plans[j][key];
                }
            }
        }
    }
}

var validatePlanJSONValues = function(policyJson, service_id, plan_id) {
    var errors = [];
    var errorCount = 0;
    
    if(typeof policyJson !== 'undefined' && policyJson !== null && typeof policyJson.schedules !== 'undefined' && policyJson.schedules !== null ){
        if( (typeof policyJson.schedules.recurring_schedule !== 'undefined' && policyJson.schedules.recurring_schedule !== null) && (policyJson.schedules.recurring_schedule.length > valueInPlan(service_id, plan_id, "recurring_schedule_count")) ){
            let error = createErrorResponse('schedules.recurring_schedule', 'policy exceeded recurring_schedule as per plan of service', plan_id, service_id)
            errors[errorCount++] = error
        }
        if( (typeof policyJson.schedules.specific_date !== 'undefined' && policyJson.schedules.specific_date !== null) && (policyJson.schedules.specific_date.length > valueInPlan(service_id, plan_id, "specific_date_count")) ){
                let error = createErrorResponse('schedules.specific_date', 'policy exceeded specific_date as per plan of service', plan_id, service_id)
                errors[errorCount++] = error
            }
    }
    if(typeof policyJson !== 'undefined' && policyJson !== null && typeof policyJson.scaling_rules !== 'undefined' && policyJson.scaling_rules !== null && (policyJson.scaling_rules.length > valueInPlan(service_id, plan_id, "scaling_rules_count")) ){
        let error = createErrorResponse('scaling_rules', 'policy exceeded scaling rules as per plan of service', plan_id, service_id)
        errors[errorCount++] = error
    }
    return errors
}

exports.validatePlan = function(policy, service_id, plan_id,  callback) {
  if(callback) {
    var errors = validatePlanJSONValues(policy, service_id, plan_id); 
    callback(errors);
  }
  else{
    logger.error('No callback function specified!', {});
    return;
  }
}