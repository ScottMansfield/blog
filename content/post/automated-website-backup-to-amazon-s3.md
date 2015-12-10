+++
title = "Automated Website Backup to Amazon S3"
date = "2012-03-24T00:12:29-05:00"
keywords = []
categories = ["PHP","Projects", "Amazon", "S3"]
+++

After reading Lifehacker's [article](http://lifehacker.com/5885392/automatically-back-up-your-web-si
te-every-night) linking to Gina Trapani's [article](http://smarterware.org/9572/automatically-back-u
p-your-web-site-every-night) about automatic website backups, I decided it would be a good idea to
implement this for my own websites. Gina's solution is great for one website, but I have multiple
websites under one user. I am definitely not a bash-fu master by any stretch of the imagination so
the best I could have done with bash would have been to copy Gina's script and modify it some to
fit my needs. I decided instead to write my backup script using PHP 5.3 (version is important!)
using the Amazon SDK  for PHP version 1.5.3. This gave me the ability to index an array by a string
if I so chose, and just, in general, felt like a more comfortable environment for me to work in.

The first few requirements from Gina's solution are exactly mine, so I quote them here:

* You're running a LAMP-based web site (Linux, Apache, MySQL and PHP/Perl/Python).
* You have command line access to your web server via SSH.
* You know how to [make new folders](https://en.wikipedia.org/wiki/Mkdir) and [chmod](https://en.wik
ipedia.org/wiki/Chmod) permissions on files.
* You're comfortable with running scripts at the command line on your server and setting up [cron](h
ttps://en.wikipedia.org/wiki/Cron)jobs.
* You know where all your web server's files are stored, what databases you need to back up, what
username and password you use to log into MySQL.

And the last one is the important change that I made to these requirements. My script will upload
the backup file to [Amazon's S3 cloud storage](https://aws.amazon.com/s3/) instead of using rsync
and ssh to upload it to another server.

* You have an Amazon account and have activated S3. You also need to have a bucket set up to receive
your backup files. You must understand that by using this script **it will cost you money** because
you are using Amazon's services. They will charge you per GB-Month you use. Please review the
pricing for S3 so you aren't taken by surprise.

## The Config Section

```php
#!/usr/local/bin/php-5.3

<?php
// the above should be changed to the path of the php 5.3
// executable on your system

error_reporting(E_ALL);

###################
### Config Section
###################
$config = array(
    'user'               => 'user',
    'path_to_sites'      => '/path/to/sites',
    'local_backup_days'  => 5,
    'home_dir'           => '/path/to/home/directory',
    's3_key'             => 'OMGTHISISMYKEY',
    's3_secret'          => 'SECRET',
    'bucket'             => 'bucket',
    'chunk_size_in_MB'   => 10,
    'remote_backup_days' => 10
);

$sites = array(
    'example.com' => array(
        'has_db'  => false),
    'blog.example.com' => array(
        'has_db'  => true,
        'db_host' => 'mysql.example.com',
        'db_name' => 'my_blog_db',
        'db_user' => 'bloguser',
        'db_pass' => 'correct horse battery staple')
    );
```

The first line is just to tell the shell that we want to run this file using the php-5.3 executable
at `/usr/local/bin/php-5.3`.  This should be changed to whatever the path of the PHP executable is
on your system, but remember that 5.3 is needed for the Amazon SDK to do its thing later on. This
hash-bang line is needed if you want to just type 

```bash
./backup_and_upload_to_s3.php
```

on the command line (or without `./` in your crontab) to run this file. In order to do this, the file must be executable, so running [cci_bash]chmod +x backup_and_upload_to_s3.php[/cci_bash] is also necessary. You could also skip these two steps and just type [cci_bash]php-5.3 backup_and_upload_to_s3.php[/cci_bash].

Next is the `$config` array for all the odds and ends that were specific to my setup.

* `user` is the name of the user running this script. It is used for naming the files as `backup_user_...`
* `path_to_sites` is the directory in which all of the website directories are placed.
* `local_backup_days` is how long to keep backups on the local system. Backups older than this are
deleted.
* `home_dir` is used for Amazon's SDK. It requires a HOME environment variable to be in place to
work.
* `s3_key` is the API key from Amazon.
* `s3_secret` is the API shared secret you also get from Amazon.
* `bucket` is the name of the Amazon bucket to upload to.
* `chunk_size_in_MB` is the size of each chunk uploaded as a multipart upload. This allows the
script to cancel an upload if an error occurs in the middle. I found 10 MB to be a good number.
* `remote_backup_days` is the number of days to keep remote backups on Amazon. Older backups are
deleted.

The `$sites` array holds all of the information about the websites that you want to back up. There
are a couple assumptions made about these websites:

* Each website resides in its own directory named the same as the website. The string put here will
be used as part of the path name after the `path_to_sites` config variable.
* Each website has a separate database. The database information is per website, so if you use on
database with table prefixes, putting the same information in of both will result in a second copy
of the whole database.

If `has_db` is false, the rest of the information is not needed, so I left it out for sites that do
not have a database. You can put as many different sites as you want in this config array and all of
them will be archived. I have about 12 sites that are all archived, some with a ton of data and some
with very little data, and all are saved.

## The Backup Process

The script will make a backup of all necessary materials to get your site up and running again after
a catastrophic event (or a host move, which can be a catastrophe in and of itself). Again, Gina says
it best:

> In order to back up your web site, your script has to back up two things: all the files that make
up the site, and all the data in your database. In this scheme you're not backing up the HTML pages
that your PHP or PERL scripts generate; you're backing up the PHP or PERL source code itself, which
accesses the data in your database. This way if your site blows up, you can restore it on a new host
and everything will work the way it does now.

### Local Backup

At the end of this portion, there will be one big `backup_username_date.bak.tar.gz` on the local
system that contains all the data for all the configured websites for that user. The script here is
rather long, so it would be best to head over to the [github repo](https://github.com/ScottMansfield
/S3-Website-Backup) I have set up with the code in it. You could even fork it and improve upon it.
If you do, I would appreciate a comment describing what you did improve.

The script first creates a directory with the date and time in the name for the backup that is
running. This will be the base temporary folder. All of the MySQL databases that are configured as
part of a site will be dumped into this folder as a gzip file. All of the websites will be tarred
and gzipped as well in a separate directory. After the two dumping / compressing phases, the whole
folder is put into another tar archive and gzipped for good measure. The temporary file is then
deleted. This all happens within the directory that the script resides.

The last step in the local backup portion is to delete older backups. The script is set up to hold
backups for the configured number of days, the default being 5. After this limit, the backups are
simply deleted. The script will output all of the information regarding what databases and
directories are being backed up and which backups are being deleted.

### Remote Backup

After the local file has been created, it is uploaded to Amazon's S3 service using the configuration
values for the bucket, key, and secret key. The file is uploaded in chinks of the size that the user
configures. The default is 10 MB, which I found to be a good balance between speed and quick
failure. The chunks are uploaded one by one to Amazon and once they are all finished, the upload is
completed. Each chunk is verified so that network failures are found out quickly. I personally also
like to have feedback regarding a long running process, so chunks are good for me.

After uploading the most recent backup, the archives older than the configured number of days are
deleted from Amazon's servers. The number of days can be configured, so please make sure you pay
attention to your budget when you select anything large. Each upload will most likely take a similar
amount of space as the one before.

## Automation

In order to automate this script, you need to add an entry into your crontab. In order to do this
type

```bash
crontab -e
```

into your console to start the crontab editing application using the default editor. Once this is
open, you need to add the script into the crontab using standard crontab syntax. The syntax is as
follows:

    * * * * * command to be executed
    - - - - -
    | | | | |
    | | | | +----- day of week (0 - 6) (Sunday=0)
    | | | +------- month (1 - 12)
    | | +--------- day of month (1 - 31)
    | +----------- hour (0 - 23)
    +------------- min (0 - 59)

The `*` in the value field above means all legal values for that column. The value column can have a
`*` or a list of elements separated by commas. An element is either a number in the ranges shown
above or two numbers in the range separated by a hyphen (meaning an inclusive range).

(Borrowed from [http://www.adminschoice.com/crontab-quick-reference](http://www.adminschoice.com/cro
ntab-quick-reference). Where would I be without Google?)

For my websites, I decided it would be good to have a daily backup performed at midnight. Thus my
crontab is as follows:

    0 0 * * * /path/to/backup/backup_and_upload_to_s3.php

This crontab is made with the assumption that I have the hash-bang line at the beginning of the file
and have run [cci]chmod[/cci] to make the file executable. Otherwise, the file will look like

    0 0 * * * php-5.3 /path/to/backup/backup_and_upload_to_s3.php

You can run the backup script as often as you'd like, but you should keep in mind that these are not
incremental backups. Each file contains all of the information that your websites contained at that
point in time. Each backup is a full, independent backup of all configured sites.

## Conclusion

Website backups are something that is often overlooked by people when on a shared environment. Some
people just assume the web host will have a backup and others will just not care. Once this solution
is set up, you can just cruise along without worrying about your websites at all. Every day a new
backup is made and uploaded to a third party whose job is to provide reliable storage. If anything
should happen to your web host, you can easily get back all of your information and be up and
running within a couple hours. Like I said above, if you have the itch to improve upon this script,
please do to [over on github](https://github.com/ScottMansfield/S3-Website-Backup). If you use it,
please drop me a line in the comments. Above all, be happy now that you don't have to worry about
your websites now that you have an automated backup in place.
