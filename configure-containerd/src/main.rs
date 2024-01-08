use std::fs;
use std::path::Path;
use toml_edit::{value, Array, Document, Item, Table, Value};

const CONTAINERD_CONFIG_PATH: &str = "/etc/containerd/config.toml";
const CONTAINERD_IMPORTS_PATH: &str = "/etc/containerd/config.toml.d";
const NYDUS_CONFIG_PATH: &str = "/etc/containerd/config.toml.d/nydus-snapshotter.toml";
const NYDUS_SOCKET_PATH: &str = "/run/containerd-nydus/containerd-nydus-grpc.sock";

fn create_nydus_config() -> Document {
    // [proxy_plugins]
    //   [proxy_plugins.nydus]
    //     type = "snapshot"
    //     address = "/run/containerd-nydus/containerd-nydus-grpc.sock"
    let mut main = Table::new();
    let mut proxy_plugins = Table::new();
    let mut nydus = Table::new();
    nydus["type"] = value("snapshot");
    nydus["address"] = value(NYDUS_SOCKET_PATH);
    proxy_plugins["nydus"] = Item::Table(nydus);
    main["proxy_plugins"] = Item::Table(proxy_plugins);
    main.into()
}

fn enable_nydus_snapshotter(doc: &mut Document) {
    if Path::new(NYDUS_CONFIG_PATH).exists() {
        panic!("{} already exists", NYDUS_CONFIG_PATH);
    }

    let nydus_config = create_nydus_config();
    if !Path::new(CONTAINERD_IMPORTS_PATH).exists() {
        fs::create_dir(CONTAINERD_IMPORTS_PATH).expect("failed to create containerd imports dir");
    }
    fs::write(NYDUS_CONFIG_PATH, nydus_config.to_string()).expect("failed to write nydus config");
    add_import(doc, NYDUS_CONFIG_PATH);

    // set disable_snapshot_annotations = false
    let entry = &mut doc["plugins"]["io.containerd.grpc.v1.cri"]["containerd"]
        ["disable_snapshot_annotations"];
    match entry {
        Item::None | Item::Value(_) => *entry = value(false),
        _ => panic!("disable_snapshot_annotations is not a value"),
    }
}

fn add_import(doc: &mut Document, path: &str) {
    let imports = &mut doc["imports"];
    match imports {
        Item::None => {
            let mut arr = Array::new();
            arr.push(path);
            *imports = value(arr);
        }
        Item::Value(val) => {
            let Value::Array(arr) = val else {
                panic!("imports is not an array");
            };
            if arr.iter().any(|x| x.as_str() == Some(NYDUS_CONFIG_PATH)) {
                panic!("{} import entry already exists", NYDUS_CONFIG_PATH);
            }
            arr.push(NYDUS_CONFIG_PATH);
        }
        _ => panic!("imports is not a value"),
    };
}

fn main() {
    let containerd_config =
        fs::read_to_string(CONTAINERD_CONFIG_PATH).expect("failed to read containerd config");
    let mut doc: Document = containerd_config.parse().expect("invalid toml document");
    enable_nydus_snapshotter(&mut doc);
    fs::write(CONTAINERD_CONFIG_PATH, doc.to_string()).expect("failed to write containerd config");
}
