A Windows test package that returns a random decimal number.
Since batch scripts don't support floating numbers a PowerShell script was used instead, and since the vSphere API executes the file in CMD a middleman batch script is needed to call the PowerShell script.
