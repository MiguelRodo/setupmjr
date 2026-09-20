run <- function(..., stdout = "", stderr = "") {
  system2("setupmjr", shQuote(c(...)), stdout = stdout, stderr = stderr)
}
