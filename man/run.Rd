\name{run}
\alias{run}
\title{Run the installed setupmjr CLI}
\usage{
run(..., stdout = "", stderr = "")
}
\arguments{
  \item{...}{Character arguments passed to the \command{setupmjr} executable.}
  \item{stdout}{Standard-output handling passed to \code{system2()}.}
  \item{stderr}{Standard-error handling passed to \code{system2()}.}
}
\value{
The value returned by \code{system2()}.
}
\description{
Runs the installed \command{setupmjr} executable without duplicating its command tree in R. The executable must be available on \env{PATH}.
}
\examples{
\dontrun{
run("--version", stdout = TRUE)
}
}
