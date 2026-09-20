library(setupmjr)

bin <- tempfile("setupmjr-bin-")
dir.create(bin)
cli <- file.path(bin, "setupmjr")
writeLines(c("#!/bin/sh", "printf '%s\\n' \"$@\""), cli)
Sys.chmod(cli, "0755")
Sys.setenv(PATH = paste(bin, Sys.getenv("PATH"), sep = .Platform$path.sep))

out <- run("git", "profile", "value with spaces", stdout = TRUE)
stopifnot(identical(out, c("git", "profile", "value with spaces")))
