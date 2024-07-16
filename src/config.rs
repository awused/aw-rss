use std::path::PathBuf;

use clap::Parser;
use once_cell::sync::Lazy;
use serde::{Deserialize, Deserializer};


#[derive(Debug, Parser)]
#[command(name = "aw-rss", about = "Awused's rss reader.")]
pub struct Opt {
    #[arg(short, long, value_parser)]
    awconf: Option<PathBuf>,

    #[arg(value_parser)]
    pub file_name: Option<PathBuf>,
}


#[derive(Debug, Deserialize, Default)]
pub struct Config {
    pub database: PathBuf,

    pub host: String,

    pub port: u16,

    #[serde(default)]
    pub log_level: String,

    #[serde(default, deserialize_with = "empty_path_is_none")]
    pub log_file: Option<PathBuf>,

    #[serde(default)]
    pub dedupe: bool,
}

// Serde seems broken with OsString for some reason
fn empty_path_is_none<'de, D, T>(deserializer: D) -> Result<Option<T>, D::Error>
where
    D: Deserializer<'de>,
    T: From<PathBuf>,
{
    let s = PathBuf::deserialize(deserializer)?;
    if s.as_os_str().is_empty() { Ok(None) } else { Ok(Some(s.into())) }
}


pub static OPTIONS: Lazy<Opt> = Lazy::new(Opt::parse);


pub static CONFIG: Lazy<Config> = Lazy::new(|| {
    match awconf::load_config::<Config>("aw-rss", OPTIONS.awconf.as_ref(), None::<&str>) {
        Ok((conf, Some(path))) => {
            info!("Loaded config from {path:?}");
            conf
        }
        Ok((conf, None)) => {
            info!("Loaded default config");
            conf
        }
        Err(e) => {
            error!("Error loading config: {e}");
            panic!("Error loading config: {e}");
        }
    }
});


pub fn init() {
    Lazy::force(&OPTIONS);
    Lazy::force(&CONFIG);
}
