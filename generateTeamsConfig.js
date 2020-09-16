
const teams = [
  {"port": 9090, "team": "fwsu", "isTrueHealthCheck": true},
  {"port": 9091, "team": "calidad"},
  {"port": 9092, "team": "platform-cancel"},
  {"port": 9093, "team": "platform-transactions"},
  {"port": 9094, "team": "platform-nonair"},
  {"port": 9095, "team": "platform-cambios"},
  {"port": 9096, "team": "platform-hoteles"},
  {"port": 9097, "team": "worklist"},
  {"port": 9098, "team": "comunicaciones"},
  {"port": 9099, "team": "reservations"},
  {"port": 9100, "team": "detail"},
  {"port": 9101, "team": "my-trips"},
  {"port": 9102, "team": "nativo"},
  {"port": 9103, "team": "agent-tools"},
  {"port": 9104, "team": "ivr"},
  {"port": 9105, "team": "encuestas"},
  {"port": 9106, "team": "soluciones"},
  {"port": 9107, "team": "cancel"},
  {"port": 9108, "team": "cambios-nonair"},
  {"port": 9109, "team": "cambios-air"}
];

function getScriptName(isTrueHc){
  return isTrueHc ? "hc.sh" : "cache-hc.sh";
}

function createTeamsConfig(teams) {
  const template = '[servers.{team}]\n bind = "0.0.0.0:{port}"\n protocol = "udp"\n balance = "roundrobin"\n backend_idle_timeout="0"\n client_idle_timeout="0"\n\n[servers.{team}.udp]\n max_requests = 1\n max_responses = 0\n\n[servers.{team}.discovery]\n kind = "static"\n static_list = [\n  "as-logstash-00:{port}",\n  "as-logstash-01:{port}"\n ]\n\n[servers.{team}.healthcheck]\n interval = "30s"\n kind = "exec"\n exec_command = "./{scriptName}"\n exec_expected_positive_output = "1"\n exec_expected_negative_output = "0"\n timeout = "5s"';
  return teams.map(t => {
    let conf;
    conf = template.replace(new RegExp("{team}","g"),t.team);
    conf = conf.replace(new RegExp("{scriptName}","g"),getScriptName(t.isTrueHealthCheck));
    return conf.replace(new RegExp("{port}","g"),t.port);
  })
  .reduce( (a,b) => a + '\n\n' + b, "" );
}

function createConfig(teams) {
  const prefix = '[api]\nenabled = true\nbind = ":9290"\n';

  return prefix + createTeamsConfig(teams);
}

console.log(createConfig(teams));