package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.hamcrest.CoreMatchers.is;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.text.DateFormat;
import java.util.ArrayList;
import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.TestConfiguration;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.hamcrest.Matchers;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
import org.quartz.Scheduler;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.SpringBootTest.WebEnvironment;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.annotation.DirtiesContext.ClassMode;
import org.springframework.test.context.junit4.SpringRunner;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.ResultActions;
import org.springframework.test.web.servlet.ResultMatcher;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.context.WebApplicationContext;

import com.fasterxml.jackson.databind.ObjectMapper;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = ClassMode.BEFORE_CLASS)
public class ScheduleRestControllerTest extends TestConfiguration {

	@MockBean
	private Scheduler scheduler;

	@Autowired
	private SpecificDateScheduleDao specificDateScheduleDao;

	@Autowired
	private MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private TestDataDbUtil testDataDbUtil;

	@Autowired
	private WebApplicationContext wac;

	private MockMvc mockMvc;

	private String appId = TestDataSetupHelper.generateAppIds(1)[0];

	@Before
	public void before() throws Exception {
		Mockito.reset(scheduler);
		testDataDbUtil.cleanupData();
		mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();

		String appId = "appId_1";
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateScheduleEntities(appId, 1);
		testDataDbUtil.insertSpecificDateSchedule(specificDateScheduleEntities);

		appId = "appId_2";
		List<RecurringScheduleEntity> recurringScheduleEntities = TestDataSetupHelper
				.generateRecurringScheduleEntities(appId, 1, 0);
		testDataDbUtil.insertRecurringSchedule(recurringScheduleEntities);

		appId = "appId_3";
		specificDateScheduleEntities = TestDataSetupHelper.generateSpecificDateScheduleEntities(appId, 2);
		testDataDbUtil.insertSpecificDateSchedule(specificDateScheduleEntities);
		recurringScheduleEntities = TestDataSetupHelper.generateRecurringScheduleEntities(appId, 1, 2);
		testDataDbUtil.insertRecurringSchedule(recurringScheduleEntities);
	}

	@Test
	public void testGetAllSchedule_with_no_schedules() throws Exception {
		ResultActions resultActions = mockMvc.perform(get(getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));

		assertNoSchedulesFound(resultActions);
	}

	@Test
	public void testGetSchedule_with_only_specificDateSchedule() throws Exception {
		String appId = "appId_1";

		ResultActions resultActions = mockMvc.perform(get(getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));

		ApplicationSchedules applicationPolicy = getApplicationSchedulesFromResultActions(resultActions);

		resultActions.andExpect(status().isOk());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));

		assertSpecificDateScheduleFoundEquals(1, appId, applicationPolicy.getSchedules().getSpecificDate());
		assertRecurringDateScheduleFoundEquals(0, appId, applicationPolicy.getSchedules().getRecurringSchedule());
	}

	@Test
	public void testGetSchedule_with_only_recurringSchedule() throws Exception {
		String appId = "appId_2";

		ResultActions resultActions = mockMvc.perform(get(getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));

		ApplicationSchedules applicationPolicy = getApplicationSchedulesFromResultActions(resultActions);

		resultActions.andExpect(status().isOk());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));

		assertSpecificDateScheduleFoundEquals(0, appId, applicationPolicy.getSchedules().getSpecificDate());
		assertRecurringDateScheduleFoundEquals(1, appId, applicationPolicy.getSchedules().getRecurringSchedule());
	}

	@Test
	public void testGetSchedule_with_specificDateSchedule_and_recurringSchedule() throws Exception {
		String appId = "appId_3";

		ResultActions resultActions = mockMvc.perform(get(getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));

		ApplicationSchedules applicationPolicy = getApplicationSchedulesFromResultActions(resultActions);

		resultActions.andExpect(status().isOk());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));

		assertSpecificDateScheduleFoundEquals(2, appId, applicationPolicy.getSchedules().getSpecificDate());
		assertRecurringDateScheduleFoundEquals(3, appId, applicationPolicy.getSchedules().getRecurringSchedule());
	}

	@Test
	public void testCreateAndGetSchedules_from_jsonFile() throws Exception {
		String policyJsonStr = getPolicyJsonContent();
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		ResultActions resultActions = mockMvc.perform(put(getSchedulerPath(appId))
				.contentType(MediaType.APPLICATION_JSON).accept(MediaType.APPLICATION_JSON).content(policyJsonStr));
		assertResponseForCreateSchedules(resultActions, status().isOk());

		resultActions = mockMvc.perform(get(getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));

		ApplicationSchedules applicationSchedules = getApplicationSchedulesFromResultActions(resultActions);
		assertSchedulesFoundEquals(applicationSchedules, appId, resultActions, 2, 4);

		Mockito.verify(scheduler, Mockito.times(6)).scheduleJob(Mockito.anyObject(), Mockito.anyObject());
	}

	@Test
	public void testCreateSchedule_with_only_specificDateSchedules() throws Exception {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String content = TestDataSetupHelper.generateJsonSchedule(2, 0);

		ResultActions resultActions = mockMvc.perform(put(getSchedulerPath(appId))
				.contentType(MediaType.APPLICATION_JSON).accept(MediaType.APPLICATION_JSON).content(content));

		assertResponseForCreateSchedules(resultActions, status().isOk());

		assertThat("It should have two specific date schedules.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(2));
		assertThat("It should have no recurring schedules.", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(0));

		Mockito.verify(scheduler, Mockito.times(2)).scheduleJob(Mockito.anyObject(), Mockito.anyObject());
	}

	@Test
	public void testCreateSchedule_with_only_recurringSchedules() throws Exception {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String content = TestDataSetupHelper.generateJsonSchedule(0, 2);

		ResultActions resultActions = mockMvc.perform(put(getSchedulerPath(appId))
				.contentType(MediaType.APPLICATION_JSON).accept(MediaType.APPLICATION_JSON).content(content));

		assertResponseForCreateSchedules(resultActions, status().isOk());

		assertThat("It should have no specific date schedules.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(0));
		assertThat("It should have two recurring schedules.",
				testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId), is(2));

		Mockito.verify(scheduler, Mockito.times(2)).scheduleJob(Mockito.anyObject(), Mockito.anyObject());
	}

	@Test
	public void testCreateSchedule_with_specificDateSchedules_and_recurringSchedules() throws Exception {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String content = TestDataSetupHelper.generateJsonSchedule(2, 2);

		ResultActions resultActions = mockMvc.perform(put(getSchedulerPath(appId))
				.contentType(MediaType.APPLICATION_JSON).accept(MediaType.APPLICATION_JSON).content(content));

		assertResponseForCreateSchedules(resultActions, status().isOk());

		assertThat("It should have two specific date schedules.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(2));
		assertThat("It should have two recurring schedules.",
				testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId), is(2));

		Mockito.verify(scheduler, Mockito.times(4)).scheduleJob(Mockito.anyObject(), Mockito.anyObject());
	}

	@Test
	public void testCreateSchedule_when_schedule_existing_for_appId() throws Exception {
		String appId = "appId_3";

		assertThat("It should have 2 specific date schedule.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(2));
		assertThat("It should have 3 recurring schedules.", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(3));

		// Create two specific date schedules and one recurring schedule for the same application.
		String content = TestDataSetupHelper.generateJsonSchedule(2, 1);
		ResultActions resultActions = mockMvc.perform(put(getSchedulerPath(appId))
				.contentType(MediaType.APPLICATION_JSON).accept(MediaType.APPLICATION_JSON).content(content));
		assertResponseForCreateSchedules(resultActions, status().isNoContent());

		assertThat("It should have 2 specific date schedules.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(2));
		assertThat("It should have 1 recurring schedule.", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(1));

		Mockito.verify(scheduler, Mockito.times(3)).scheduleJob(Mockito.anyObject(), Mockito.anyObject());
		Mockito.verify(scheduler, Mockito.times(10)).deleteJob(Mockito.anyObject());
	}

	@Test
	public void testCreateSchedule_without_appId() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);
		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put("/v2/schedules").contentType(MediaType.APPLICATION_JSON).content(content));

		resultActions.andExpect(status().isNotFound());

	}

	@Test
	public void testCreateSchedule_without_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.getSchedules().setTimeZone(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(getSchedulerPath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.value.not.specified.timezone",
				"timeZone");
		assertErrorMessages(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_empty_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.getSchedules().setTimeZone("");

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(getSchedulerPath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.value.not.specified.timezone",
				"timeZone");
		assertErrorMessages(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_invalid_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.getSchedules().setTimeZone("Invalid Timezone");

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(getSchedulerPath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.invalid.timezone", "timeZone");
		assertErrorMessages(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_defaultInstanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.setInstanceMinCount(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(getSchedulerPath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.not.specified",
				"instance_min_count");
		assertErrorMessages(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.setInstanceMaxCount(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(getSchedulerPath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.not.specified",
				"instance_max_count");
		assertErrorMessages(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_negative_defaultInstanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);
		int instanceMinCount = -1;
		applicationPolicy.setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(getSchedulerPath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.invalid",
				"instance_min_count", instanceMinCount);
		assertErrorMessages(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_negative_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);
		int instanceMaxCount = -1;
		applicationPolicy.setInstanceMaxCount(instanceMaxCount);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(getSchedulerPath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.invalid",
				"instance_max_count", instanceMaxCount);
		assertErrorMessages(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_defaultInstanceMinCount_greater_than_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		Integer instanceMinCount = 5;
		Integer instanceMaxCount = 1;
		applicationPolicy.setInstanceMaxCount(instanceMaxCount);
		applicationPolicy.setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(getSchedulerPath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage(
				"data.default.instanceCount.invalid.min.greater", "instance_max_count", instanceMaxCount,
				"instance_min_count", instanceMinCount);
		assertErrorMessages(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_instanceMaxAndMinCount_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.setInstanceMaxCount(null);
		applicationPolicy.setInstanceMinCount(null);
		applicationPolicy.getSchedules().setTimeZone(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(getSchedulerPath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		List<String> messages = new ArrayList<>();
		messages.add(
				messageBundleResourceHelper.lookupMessage("data.default.value.not.specified", "instance_min_count"));
		messages.add(
				messageBundleResourceHelper.lookupMessage("data.default.value.not.specified", "instance_max_count"));
		messages.add(messageBundleResourceHelper.lookupMessage("data.value.not.specified.timezone", "timeZone"));

		assertErrorMessages(resultActions, messages.toArray(new String[0]));
	}

	@Test
	public void testCreateSchedule_multiple_error() throws Exception {
		// Should be individual each test.

		testCreateSchedule_negative_defaultInstanceMaxCount();

		testCreateSchedule_without_defaultInstanceMinCount();

		testCreateSchedule_without_defaultInstanceMaxCount();

		testCreateSchedule_defaultInstanceMinCount_greater_than_defaultInstanceMaxCount();
	}

	@Test
	public void testDeleteSchedule_with_only_specificDateSchedule() throws Exception {
		String appId = "appId_1";

		assertThat("It should have 1 specific date schedule.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(1));
		assertThat("It should have no recurring schedules.", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(0));

		ResultActions resultActions = mockMvc
				.perform(delete(getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));
		assertSchedulesAreDeleted(resultActions);

		assertThat("It should have no specific date schedules.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(0));
		assertThat("It should have no recurring schedules.", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(0));

		Mockito.verify(scheduler, Mockito.times(2)).deleteJob(Mockito.anyObject());
	}

	@Test
	public void testDeleteSchedule_with_only_recurringSchedule() throws Exception {
		String appId = "appId_2";

		assertThat("It should have no specific date schedules.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(0));
		assertThat("It should have 1 recurring schedule.", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(1));

		ResultActions resultActions = mockMvc
				.perform(delete(getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));
		assertSchedulesAreDeleted(resultActions);

		assertThat("It should have no specific date schedules.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(0));
		assertThat("It should have no recurring schedules.", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(0));

		Mockito.verify(scheduler, Mockito.times(2)).deleteJob(Mockito.anyObject());
	}

	@Test
	public void testDeleteSchedule_with_specificDateSchedule_and_recurringSchedule() throws Exception {
		String appId = "appId_3";

		assertThat("It should have 2 specific date schedule.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(2));
		assertThat("It should have 3 recurring schedule.", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(3));

		ResultActions resultActions = mockMvc
				.perform(delete(getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));
		assertSchedulesAreDeleted(resultActions);

		assertThat("It should have no specific date schedules.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(0));
		assertThat("It should have no recurring schedules.", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(0));

		Mockito.verify(scheduler, Mockito.times(10)).deleteJob(Mockito.anyObject());
	}

	@Test
	public void testDeleteSchedules_appId_without_schedules() throws Exception {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];

		assertThat("It should have no specific date schedules.",
				testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId), is(0));
		assertThat("It should have no recurring schedules.", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(0));

		assertThat("It should have 3 specific date schedules.", testDataDbUtil.getNumberOfSpecificDateSchedules(),
				is(3));
		assertThat("It should have 4 recurring schedules.", testDataDbUtil.getNumberOfRecurringSchedules(), is(4));

		ResultActions resultActions = mockMvc
				.perform(delete(getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));
		assertNoSchedulesFound(resultActions);

		assertThat("It should have 3 specific date schedules.", testDataDbUtil.getNumberOfSpecificDateSchedules(),
				is(3));
		assertThat("It should have 4 recurring schedules.", testDataDbUtil.getNumberOfRecurringSchedules(), is(4));

		Mockito.verify(scheduler, Mockito.never()).deleteJob(Mockito.anyObject());
	}

	private void assertNoSchedulesFound(ResultActions resultActions) throws Exception {
		resultActions.andExpect(content().string(Matchers.isEmptyString()));
		resultActions.andExpect(header().doesNotExist("Content-type"));
		resultActions.andExpect(status().isNotFound());
	}

	private void assertResponseForCreateSchedules(ResultActions resultActions, ResultMatcher expectedStatus)
			throws Exception {
		resultActions.andExpect(content().string(Matchers.isEmptyString()));
		resultActions.andExpect(header().doesNotExist("Content-type"));
		resultActions.andExpect(expectedStatus);
	}

	private void assertSchedulesFoundEquals(ApplicationSchedules applicationSchedules, String appId,
			ResultActions resultActions, int expectedSpecificDateSchedulesTobeFound,
			int expectedRecurringSchedulesTobeFound) throws Exception {

		resultActions.andExpect(status().isOk());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));

		assertSpecificDateScheduleFoundEquals(expectedSpecificDateSchedulesTobeFound, appId,
				applicationSchedules.getSchedules().getSpecificDate());
		assertRecurringDateScheduleFoundEquals(expectedRecurringSchedulesTobeFound, appId,
				applicationSchedules.getSchedules().getRecurringSchedule());
	}

	private void assertSpecificDateScheduleFoundEquals(int expectedSchedulesTobeFound, String expectedAppId,
			List<SpecificDateScheduleEntity> specificDateScheduls) {
		if (specificDateScheduls == null) {
			assertEquals(expectedSchedulesTobeFound, 0);
		} else {
			assertEquals(expectedSchedulesTobeFound, specificDateScheduls.size());
			for (ScheduleEntity entity : specificDateScheduls) {
				assertEquals(expectedAppId, entity.getAppId());
			}
		}
	}

	private void assertRecurringDateScheduleFoundEquals(int expectedRecurringSchedulesTobeFound, String expectedAppId,
			List<RecurringScheduleEntity> recurring_schedule) {
		if (recurring_schedule == null) {
			assertEquals(expectedRecurringSchedulesTobeFound, 0);
		} else {
			assertEquals(expectedRecurringSchedulesTobeFound, recurring_schedule.size());
			for (ScheduleEntity entity : recurring_schedule) {
				assertEquals(expectedAppId, entity.getAppId());
			}
		}

	}

	private void assertErrorMessages(ResultActions resultActions, String... expectedErrorMessages) throws Exception {
		resultActions.andExpect(jsonPath("$").value(Matchers.containsInAnyOrder(expectedErrorMessages)));
		resultActions.andExpect(jsonPath("$").isArray());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));
		resultActions.andExpect(status().isBadRequest());
	}

	private void assertSchedulesAreDeleted(ResultActions resultActions) throws Exception {
		resultActions.andExpect(content().string(Matchers.isEmptyString()));
		resultActions.andExpect(header().doesNotExist("Content-type"));
		resultActions.andExpect(status().isNoContent());
	}

	private String getSchedulerPath(String appId) {
		return String.format("/v2/schedules/%s", appId);
	}

	private ApplicationSchedules getApplicationSchedulesFromResultActions(ResultActions resultActions)
			throws IOException {
		ObjectMapper mapper = new ObjectMapper();
		mapper.setDateFormat(DateFormat.getDateInstance(DateFormat.LONG));
		return mapper.readValue(resultActions.andReturn().getResponse().getContentAsString(),
				ApplicationSchedules.class);
	}

	public static String getPolicyJsonContent() {
		BufferedReader br = new BufferedReader(
				new InputStreamReader(ApplicationSchedules.class.getResourceAsStream("/fakePolicy.json")));
		String tmp;
		String jsonPolicyStr = "";
		try {
			while ((tmp = br.readLine()) != null) {
				jsonPolicyStr += tmp;
			}
		} catch (IOException e) {
			e.printStackTrace();
		}
		jsonPolicyStr = jsonPolicyStr.replaceAll("\\s+", " ");
		return jsonPolicyStr;
	}
}
