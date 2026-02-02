# Device configurations

When rebuilding the cache, add getting device configurations.  The following sections call out each API endpoint.  This enhancement is similar to [the cache enhancement](cache/CACHE_ENHANCEMENT.md) previously done, and this document follows a similar format.

# devices

All devices (ap, switch, and gateway) start with the same json definition and are then described through $ref.  This starting schema should help define the Structs needed to parse the API data.

For every device from each type in the cache inventory (.inventory.ap[*], .inventory.switch[*], .inventory.gateway[*]) check if `site_id` is non-null and then use it with the `id` with the following API call to retrieve the device configuration.  If a HTTP/200 reply is returned from getting the devices, save the returned json config in .configs.ap[*], .configs.switch[*], and .configs.gateway[*].  PLease reference [JSON_SCHEMA_REFERENCE_CONTEXT.md](./docs/JSON_SCHEMA_REFERENCE_CONTEXT.md) for how to read the schema files.

Map lookups should be also be created at the start of each app execution that maps the device Name and MAC to a pointer to the json config of that device.

- URL: GET /api/v1/sites/{site_id}/devices/{device_id}
- Schema:
  - [sites_site_id_devices_device_id_get.json](./schemas/sites_site_id_devices_device_id_get.json)
- Cache Path:
  - cache.config.ap[*]
  - cache.config.switch[*]
  - cache.config.gateway[*]
- Index
  - map[string]*APIConfig by Name
  - map[string]*APICOnfig by MAC
- Command
  - show api config [<site_name>] [<mac>|<device_name>]
