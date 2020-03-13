import { Component, OnInit, ViewChild, ElementRef } from '@angular/core';
import { SampleGuide } from '../models/sample-guide.model'
import { Tree } from '../models/tree.model'

import 'codemirror/mode/javascript/javascript.js';

declare const $: any;
// declare let ace: any;

@Component({
  selector: 'app-micro-guide',
  templateUrl: './micro-guide.component.html',
  styleUrls: ['./micro-guide.component.css']
})
export class MicroGuideComponent implements OnInit {
  @ViewChild('explorer') explorer:ElementRef;
  text: string = "";
  sampleGuide: SampleGuide = new SampleGuide();
  tree: Tree = new Tree();
  searchName: string = "";
  codeconfig = {lineNumbers: true, theme: 'darcula', mode: "javascript", readOnly: true};
  // CF_INSTANCE_INTERNAL_IP = '${CF_INSTANCE_INTERNAL_IP}';
  // PORT = '${PORT}';
  java_version = `
<parent>
  <groupId>org.springframework.boot</groupId>
  <artifactId>spring-boot-starter-parent</artifactId>
  <version>1.5.13.RELEASE</version>
  <relativePath/>
</parent>
	
<properties>
  <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
  <project.reporting.outputEncoding>UTF-8</project.reporting.outputEncoding>
  <java.version>1.8</java.version>
  <spring-cloud.version>Edgware.RELEASE</spring-cloud.version>
</properties>  
  `

  first_pom =  `
<dependencies>
  <!-- required start -->git
  <!--spring cloud-->
  <dependency>
    <groupId>org.springframework.cloud</groupId>
    <artifactId>spring-cloud-starter-config</artifactId>
  </dependency>
  <dependency>
    <groupId>org.springframework.cloud</groupId>
    <artifactId>spring-cloud-starter-eureka</artifactId>
  </dependency>
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-actuator</artifactId>
  </dependency>
  <dependency>
    <groupId>org.springframework.cloud</groupId>
    <artifactId>spring-cloud-starter-hystrix</artifactId>
  </dependency>
  
  <!-- spring-boot-starter-test -->
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-test</artifactId>
    <scope>test</scope>
  </dependency>
  
  <!-- swagger api -->
  <dependency>
    <groupId>io.springfox</groupId>
    <artifactId>springfox-swagger2</artifactId>
    <version>2.3.1</version>
  </dependency>
  <dependency>
    <groupId>io.springfox</groupId>
    <artifactId>springfox-swagger-ui</artifactId>
    <version>2.3.1</version>
  </dependency>

  <!-- spring cloud connector -->
  <dependency>
    <groupId>io.pivotal.spring.cloud</groupId>
    <artifactId>spring-cloud-services-cloudfoundry-connector</artifactId>
    <version>1.6.1.RELEASE</version>
  </dependency>
  <dependency>
    <groupId>io.pivotal.spring.cloud</groupId>
    <artifactId>spring-cloud-services-spring-connector</artifactId>
    <version>1.6.1.RELEASE</version>
  </dependency>  
  <!-- required end -->
</dependencies>
  `
  mysql_pom = `
<dependencies>
  <!-- required start -->
  .....
  <!-- required end -->
  
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-cloud-connectors</artifactId>
  </dependency>
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-data-jpa</artifactId>
  </dependency>
  <dependency>
    <groupId>mysql</groupId>
    <artifactId>mysql-connector-java</artifactId>
    <scope>runtime</scope>
  </dependency>
</dependencies>  
  `
  redis_pom = `
<dependencies>
  <!-- required start -->
  .....
  <!-- required end -->
  
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-cloud-connectors</artifactId>
  </dependency>
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-data-redis</artifactId>
  </dependency>
  <dependency>
    <groupId>org.springframework.session</groupId>
    <artifactId>spring-session</artifactId>
  </dependency>
</dependencies>  
  `

  rabbitmq_pom = `
<dependencies>
  <!-- required start -->
  .....
  <!-- required end -->
  
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-amqp</artifactId>
  </dependency>
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-cloud-connectors</artifactId>
  </dependency>
</dependencies>  
  `
  first_java = `
package com.crossent.microservice;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.cloud.client.circuitbreaker.EnableCircuitBreaker;
import org.springframework.cloud.client.discovery.EnableDiscoveryClient;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import springfox.documentation.builders.PathSelectors;
import springfox.documentation.builders.RequestHandlerSelectors;
import springfox.documentation.spi.DocumentationType;
import springfox.documentation.spring.web.plugins.Docket;
import springfox.documentation.swagger2.annotations.EnableSwagger2;

@EnableDiscoveryClient
@EnableCircuitBreaker
@SpringBootApplication
public class FrontApplication {

	public static void main(String[] args) {
		SpringApplication.run(FrontApplication.class, args);
	}

}
  
@Configuration
@EnableSwagger2
class SwaggerConfig {
    @Bean
    public Docket api() {
        return new Docket(DocumentationType.SWAGGER_2)
                .select().apis(RequestHandlerSelectors.any())
                .paths(PathSelectors.ant("/api/**"))
                .build();
    }
}  
  `

  application_properties = `
server.port: \${PORT:8081}
eureka.instance.hostname=\${CF_INSTANCE_INTERNAL_IP:localhost}
eureka.instance.nonSecurePort=\${PORT:8091}
`

  mysql_config_java = `
package com.crossent.microservice;

import org.springframework.cloud.config.java.AbstractCloudConfig;
import org.springframework.cloud.config.java.ServiceScan;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Profile;

import javax.sql.DataSource;

@Configuration
@ServiceScan
@Profile("cloud")
public class DbConfig extends AbstractCloudConfig {
}  
  `

  redis_config_java = `
package com.crossent.microservice;

import org.springframework.cloud.config.java.AbstractCloudConfig;
import org.springframework.cloud.config.java.ServiceScan;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Profile;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.session.data.redis.config.ConfigureRedisAction;

@Configuration
@ServiceScan
@Profile("cloud")
public class RedisConfig extends AbstractCloudConfig {

    @Bean
    public RedisTemplate redisTemplate() {
        return new StringRedisTemplate(connectionFactory().redisConnectionFactory());
    }

    @Bean
    public static ConfigureRedisAction configureRedisAction() {
        return ConfigureRedisAction.NO_OP;
    }
}  
  `

  rabbitmq_config_java = `
package com.crossent.microservice;

import org.springframework.amqp.rabbit.core.RabbitTemplate;

import org.springframework.cloud.config.java.AbstractCloudConfig;
import org.springframework.cloud.config.java.ServiceScan;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Profile;

@Configuration
@ServiceScan
public class RabbitConfig extends AbstractCloudConfig {

    @Bean
    public RabbitTemplate rabbitTemplate() {
        return new RabbitTemplate(connectionFactory().rabbitConnectionFactory());
    }

}  
  `
  mysql_front1_java = `
package com.crossent.microservice.web;

import com.crossent.microservice.domain.User;
import com.crossent.microservice.service.FooService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
public class FrontController {

	private static final Logger logger = LoggerFactory.getLogger(FrontController.class);

	@Autowired
	private FooService fooService;

	@RequestMapping("/list")
	@ResponseBody
	public List<User> users() {
		return fooService.users();
	}

	@GetMapping(path="/add")
	@ResponseBody
	public String addNewUser (@RequestParam String name, @RequestParam String email) {
		User n = new User();
		n.setName(name);
		n.setEmail(email);
		fooService.save(n);

		return "Saved";
	}
}
  `

  mysql_front2_java = `
package com.crossent.microservice.service;

import com.crossent.microservice.domain.User;
import com.crossent.microservice.repository.FooRepository;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;

@Service
@Transactional
public class FooService {
    @Autowired
    private FooRepository fooRepository;

    public List<User> users() {
        return fooRepository.findAll();
    }

    public void save(User n) {
        fooRepository.save(n);
    }
}
  `

  mysql_front3_java = `
package com.crossent.microservice.repository;

import com.crossent.microservice.domain.User;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface FooRepository extends JpaRepository<User, Long> {
}
  `

  mysql_front4_java = `
package com.crossent.microservice.domain;

import javax.persistence.Entity;
import javax.persistence.GeneratedValue;
import javax.persistence.GenerationType;
import javax.persistence.Id;

@Entity
public class User {
    @Id
    @GeneratedValue(strategy=GenerationType.AUTO)
    private Long id;

    private String name;

    private String email;

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getEmail() {
        return email;
    }

    public void setEmail(String email) {
        this.email = email;
    }
}
  `


  redis_front_java = `
package com.crossent.microservice;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.ValueOperations;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;
import org.springframework.data.redis.core.StringRedisTemplate;

@Controller
public class FrontController {

	private static final Logger logger = LoggerFactory.getLogger(FrontController.class);

	@Autowired
	private StringRedisTemplate template;

	@RequestMapping("/test")
	@ResponseBody
	public String test() {
		ValueOperations<String, String> ops = this.template.opsForValue();
		String key = "spring.boot.redis.test";
		if (!this.template.hasKey(key)) {
			ops.set(key, "foo");
		}
		System.out.println("Found key " + key + ", value=" + ops.get(key));
		return "test";
	}
}
  `

  rabbitmq_front1_java = `
package com.crossent.microservice;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;

@Controller
public class FrontController {

	private static final Logger logger = LoggerFactory.getLogger(FrontController.class);

	@Autowired
	Sender sender;

	@RequestMapping("/test")
	@ResponseBody
	public String test() {
		sender.send("Hello Messsaging...");
		return "test";
	}
}
  `

  rabbitmq_front2_java = `
package com.crossent.microservice;

import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.stereotype.Component;

@Component
public class Receiver {
    @RabbitListener(queues = "TestMQ")
    public void processMessage(String content){
        System.out.println(">>>>>>" + content);
    }
}
  `

  rabbitmq_front3_java = `
package com.crossent.microservice;

import org.springframework.amqp.core.Queue;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Bean;
import org.springframework.stereotype.Component;

@Component
public class Sender {
    @Autowired
    RabbitTemplate template;

    @Bean
    Queue queue(){
        return new Queue("TestMQ", false);
    }

    public void send(String message) {
        template.convertAndSend("TestMQ", message);
    }
}
  `

  constructor() { }

  ngOnInit() {
    $('.menu .item').tab({});

    $('#explorer').jstree(this.tree.dirs);


    $(this.explorer.nativeElement).on('click', (e) => {
      //console.log(e)
      const src: string = e.target.innerText;
      const id: string = e.target.id;

      if(src == 'FrontApplication.java'){
        this.text = this.sampleGuide.frontApplication;
      }else if(src == 'FrontController.java') {
        this.text = this.sampleGuide.frontCongroller;
      }else if(src == 'BackApplication.java'){
        this.text = this.sampleGuide.backApplication;
      }else if(src == 'BackController.java'){
        this.text = this.sampleGuide.backCongroller;
      }else if(src == 'Data.java'){
        if(id == 'front_data_anchor'){
          this.text = this.sampleGuide.frontData;
        }else{
          this.text = this.sampleGuide.backData;
        }
      }else if(src == 'pom.xml'){
        if(id == 'front_pom_anchor'){
          this.text = this.sampleGuide.frontPom;
        }else{
          this.text = this.sampleGuide.backPom;
        }
      }else if(src == 'manifest.yml'){
        if(id == 'front_manfiest'){
          this.text = this.sampleGuide.frontManifest;
        }else{
          this.text = this.sampleGuide.backManifest;
        }
      }else if(src == 'application.properties'){
        if(id == 'front_application_properties_anchor'){
          this.text = this.sampleGuide.frontApplicationProperties;
        }else{
          this.text = this.sampleGuide.backApplicationProperties;
        }
      }else if(src == 'bootstrap.properties'){
        if(id == 'front_bootstrap_properties_anchor'){
          this.text = this.sampleGuide.frontBootstrapProperties;
        }else{
          this.text = this.sampleGuide.backBootstrapProperties;
        }
      }else if(src == 'index.html'){
        this.text = this.sampleGuide.indexHtml;
      }else if(src == 'albums.js'){
        this.text = this.sampleGuide.albumsJs;
      }else if(src == 'app.js'){
        this.text = this.sampleGuide.appJs;
      }else if(src == 'errors.js'){
        this.text = this.sampleGuide.errorsJs;
      }else if(src == 'albums.html'){
        this.text = this.sampleGuide.albumsHtml;
      }else if(src == 'errors.html'){
        this.text = this.sampleGuide.errorsHtml;
      }else{
        this.text = "";
      }
    });
  }


}
