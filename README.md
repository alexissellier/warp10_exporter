## Warp10 exporter for Prometheus [![Go Report Card](https://goreportcard.com/badge/github.com/AlexisSellier/warp10_exporter)](https://goreportcard.com/report/github.com/AlexisSellier/warp10_exporter)

Scrape the sensision output of a [Warp10](https://github.com/cityzendata/warp10-platform) daemon (currently handling version 1.0.18)

This exporter is mostly to know what happen when you are building your Warp10 stack and can't monitore Warp10 itself :)

### Adding more metrics
The list of all metrics can be found [here](https://github.com/cityzendata/warp10-platform/blob/master/warp10/src/main/java/io/warp10/continuum/sensision/SensisionConstants.java)

### Next step

Next steps are :

* Handle labels parsing
* Refactor the parsing by using a Scanner
* Add a configuration file to specify only a few metrics
