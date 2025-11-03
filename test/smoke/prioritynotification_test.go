package smoke

import (
	"testing"

	"github.com/target/goalert/test/smoke/harness"
)

func TestPriorityBasedNotificationRouting(t *testing.T) {
	t.Parallel()

	sql := `
	insert into users (id, name, email) 
	values 
		({{uuid "u1"}}, 'bob', 'bob@example.com');
	
	insert into user_contact_methods (id, user_id, name, type, value) 
	values
		({{uuid "c1"}}, {{uuid "u1"}}, 'sms', 'SMS', {{phone "1"}}),
		({{uuid "c2"}}, {{uuid "u1"}}, 'voice', 'VOICE', {{phone "2"}});

	-- High priority (1-3) -> Voice immediately
	insert into user_notification_rules (user_id, contact_method_id, delay_minutes, conditions) 
	values
		({{uuid "u1"}}, {{uuid "c2"}}, 0, '{"metadata": {"priority": {"min": 1, "max": 3}}}'::jsonb);

	-- Medium priority (4-6) -> SMS immediately
	insert into user_notification_rules (user_id, contact_method_id, delay_minutes, conditions) 
	values
		({{uuid "u1"}}, {{uuid "c1"}}, 0, '{"metadata": {"priority": {"min": 4, "max": 6}}}'::jsonb);

	insert into escalation_policies (id, name) 
	values 
		({{uuid "e1"}}, 'esc policy');
	
	insert into escalation_policy_steps (id, escalation_policy_id) 
	values 
		({{uuid "es1"}}, {{uuid "e1"}});
	
	insert into escalation_policy_actions (escalation_policy_step_id, user_id) 
	values 
		({{uuid "es1"}}, {{uuid "u1"}});

	insert into services (id, escalation_policy_id, name) 
	values
		({{uuid "s1"}}, {{uuid "e1"}}, 'service');
	`
	h := harness.NewHarness(t, sql, "ids-to-uuids")
	defer h.Close()

	tw := h.Twilio(t)
	sms := tw.Device(h.Phone("1"))
	voice := tw.Device(h.Phone("2"))

	h.GraphQLQuery2(`mutation{createAlert(input:{serviceID:"` + h.UUID("s1") + `",summary:"high priority alert",meta:[{key:"priority", value: "2"}]}){id}}`)
	voice.ExpectVoice("high priority alert")

	h.GraphQLQuery2(`mutation{createAlert(input:{serviceID:"` + h.UUID("s1") + `",summary:"medium priority alert",meta:[{key:"priority", value: "5"}]}){id}}`)
	sms.ExpectSMS("medium priority alert")
}

func TestPriorityNotificationBackwardCompatibility(t *testing.T) {
	t.Parallel()

	sql := `
	insert into users (id, name, email) 
	values 
		({{uuid "u1"}}, 'bob', 'bob@example.com');
	
	insert into user_contact_methods (id, user_id, name, type, value) 
	values
		({{uuid "c1"}}, {{uuid "u1"}}, 'sms', 'SMS', {{phone "1"}});

	-- Rule without conditions (NULL) should match all alerts
	insert into user_notification_rules (user_id, contact_method_id, delay_minutes) 
	values
		({{uuid "u1"}}, {{uuid "c1"}}, 0);

	insert into escalation_policies (id, name) 
	values 
		({{uuid "e1"}}, 'esc policy');
	
	insert into escalation_policy_steps (id, escalation_policy_id) 
	values 
		({{uuid "es1"}}, {{uuid "e1"}});
	
	insert into escalation_policy_actions (escalation_policy_step_id, user_id) 
	values 
		({{uuid "es1"}}, {{uuid "u1"}});

	insert into services (id, escalation_policy_id, name) 
	values
		({{uuid "s1"}}, {{uuid "e1"}}, 'service');
	`
	h := harness.NewHarness(t, sql, "ids-to-uuids")
	defer h.Close()

	tw := h.Twilio(t)
	sms := tw.Device(h.Phone("1"))

	h.GraphQLQuery2(`mutation{createAlert(input:{serviceID:"` + h.UUID("s1") + `",summary:"alert with priority",meta:[{key:"priority", value: "5"}]}){id}}`)
	sms.ExpectSMS("alert with priority")

	h.CreateAlert(h.UUID("s1"), "alert without priority")
	sms.ExpectSMS("alert without priority")
}

func TestPriorityNotificationEqualsOperator(t *testing.T) {
	t.Parallel()

	sql := `
	insert into users (id, name, email) 
	values 
		({{uuid "u1"}}, 'bob', 'bob@example.com');
	
	insert into user_contact_methods (id, user_id, name, type, value) 
	values
		({{uuid "c1"}}, {{uuid "u1"}}, 'sms', 'SMS', {{phone "1"}});

	-- Only match severity = "critical"
	insert into user_notification_rules (user_id, contact_method_id, delay_minutes, conditions) 
	values
		({{uuid "u1"}}, {{uuid "c1"}}, 0, '{"metadata": {"severity": {"equals": "critical"}}}'::jsonb);

	insert into escalation_policies (id, name) 
	values 
		({{uuid "e1"}}, 'esc policy');
	
	insert into escalation_policy_steps (id, escalation_policy_id) 
	values 
		({{uuid "es1"}}, {{uuid "e1"}});
	
	insert into escalation_policy_actions (escalation_policy_step_id, user_id) 
	values 
		({{uuid "es1"}}, {{uuid "u1"}});

	insert into services (id, escalation_policy_id, name) 
	values
		({{uuid "s1"}}, {{uuid "e1"}}, 'service');
	`
	h := harness.NewHarness(t, sql, "ids-to-uuids")
	defer h.Close()

	tw := h.Twilio(t)
	sms := tw.Device(h.Phone("1"))

	h.GraphQLQuery2(`mutation{createAlert(input:{serviceID:"` + h.UUID("s1") + `",summary:"critical alert",meta:[{key:"severity", value: "critical"}]}){id}}`)
	sms.ExpectSMS("critical alert")

	h.GraphQLQuery2(`mutation{createAlert(input:{serviceID:"` + h.UUID("s1") + `",summary:"warning alert",meta:[{key:"severity", value: "warning"}]}){id}}`)
	sms.IgnoreUnexpectedSMS("warning alert")
}

func TestPriorityNotificationMinMaxOperators(t *testing.T) {
	t.Parallel()

	sql := `
	insert into users (id, name, email) 
	values 
		({{uuid "u1"}}, 'bob', 'bob@example.com');
	
	insert into user_contact_methods (id, user_id, name, type, value) 
	values
		({{uuid "c1"}}, {{uuid "u1"}}, 'sms', 'SMS', {{phone "1"}}),
		({{uuid "c2"}}, {{uuid "u1"}}, 'voice', 'VOICE', {{phone "2"}});

	-- Priority >= 5 (min only)
	insert into user_notification_rules (user_id, contact_method_id, delay_minutes, conditions) 
	values
		({{uuid "u1"}}, {{uuid "c1"}}, 0, '{"metadata": {"priority": {"min": 5}}}'::jsonb);

	-- Priority <= 3 (max only)
	insert into user_notification_rules (user_id, contact_method_id, delay_minutes, conditions) 
	values
		({{uuid "u1"}}, {{uuid "c2"}}, 0, '{"metadata": {"priority": {"max": 3}}}'::jsonb);

	insert into escalation_policies (id, name) 
	values 
		({{uuid "e1"}}, 'esc policy');
	
	insert into escalation_policy_steps (id, escalation_policy_id) 
	values 
		({{uuid "es1"}}, {{uuid "e1"}});
	
	insert into escalation_policy_actions (escalation_policy_step_id, user_id) 
	values 
		({{uuid "es1"}}, {{uuid "u1"}});

	insert into services (id, escalation_policy_id, name) 
	values
		({{uuid "s1"}}, {{uuid "e1"}}, 'service');
	`
	h := harness.NewHarness(t, sql, "ids-to-uuids")
	defer h.Close()

	tw := h.Twilio(t)
	sms := tw.Device(h.Phone("1"))
	voice := tw.Device(h.Phone("2"))

	h.GraphQLQuery2(`mutation{createAlert(input:{serviceID:"` + h.UUID("s1") + `",summary:"priority 7 alert",meta:[{key:"priority", value: "7"}]}){id}}`)
	sms.ExpectSMS("priority 7 alert")

	h.GraphQLQuery2(`mutation{createAlert(input:{serviceID:"` + h.UUID("s1") + `",summary:"priority 2 alert",meta:[{key:"priority", value: "2"}]}){id}}`)
	voice.ExpectVoice("priority 2 alert")

	h.GraphQLQuery2(`mutation{createAlert(input:{serviceID:"` + h.UUID("s1") + `",summary:"priority 4 alert",meta:[{key:"priority", value: "4"}]}){id}}`)
	sms.IgnoreUnexpectedSMS("priority 4 alert")
	voice.IgnoreUnexpectedVoice("priority 4 alert")
}
