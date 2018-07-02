//@flow

const server_ip = process.env.REACT_APP_FLOWER_SERVER_IP || "0.0.0.0";
const base_url = "http://" + server_ip + ":5000/";

export function fetchFlows(filters: *, then: (*) => mixed) {
    var filter_object = {};

    if (hasNotEmpty(filters, "text_filter"))
        filter_object["flow.data"] = filters["text_filter"];

    if (hasNotEmpty(filters, "dst_ip") && hasNotEmpty(filters, "dst_port")) {
        filter_object["dst_ip"] = filters["dst_ip"];
        filter_object["dst_port"] = filters["dst_port"];
    }
    if (hasNotEmpty(filters, "from_time") && hasNotEmpty(filters, "to_time")) {
        filter_object["from_time"] = filters["from_time"];
        filter_object["to_time"] = filters["to_time"];
    }
    if (hasNotEmpty(filters, "starred"))
        filter_object["starred"] = filters["starred"];

    console.log("Fetching flows: ");
    console.log(filter_object);

    fetch(base_url + "query", {
        method: "POST",
        headers: {
            Accept: "application/json",
            "Content-Type": "application/json"
        },
        body: JSON.stringify(filter_object)
    })
        .then(response => {
            console.log(response);
            return response.json();
        })
        .then(responseJson => {
            then(responseJson);
        })
        .catch(error => {
            console.log("errore con ws");
            console.error(error);
        });
}

export function fetchServices(then: (*) => mixed) {
    console.log("FETCHING services!!");
    return fetchUrl(base_url + "services", data => then(data.sort()));
}

export function fetchFiles(then: (*) => mixed) {
    return fetchUrl(base_url + "files", data => then(data.sort().reverse()));
}
function hasNotEmpty(object, key) {
    return object && key in object && object[key];
}

export function fetchFlow(flow_id: string, then: (*) => mixed) {
    var url = base_url + "flow/" + flow_id;
    return fetchUrl(url, then);
}

function fetchUrl(url, then) {
    return fetch(url)
        .then(response => response.json())
        .then(responseJson => {
            then(responseJson);
        })
        .catch(error => {
            console.log("errore con ws");
            console.error(error);
        });
}

export function setStarred(flow_id: string, star: boolean) {
    fetch(base_url + "star/" + flow_id + "/" + (star ? 1 : 0));
}

export function getPythonRequest(request: string, then: (*) => mixed) {
    fetch(base_url + "to_python_request/1", {
        method: "POST",
        body: request
    })
        .then(response => {
            console.log(response);
            return response.text();
        })
        .then(responseText => {
            then(responseText);
        });
}
export function getPwnRequest(flow_id: string, then: (*) => mixed) {
    fetch(base_url + "to_pwn/" + flow_id)
        .then(response => {
            console.log(response);
            return response.text();
        })
        .then(responseText => {
            then(responseText);
        });
}
