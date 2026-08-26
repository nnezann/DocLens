import type {ReactNode} from 'react';
import Heading from '@theme/Heading';

const FeatureList = [
  {
    title: 'Architecture',
    description: 'Understand service boundaries, ownership, communication, and data flow.',
  },
  {
    title: 'Contracts',
    description: 'Find REST, protobuf/gRPC, and RabbitMQ interfaces as they are formalized.',
  },
  {
    title: 'Operations',
    description: 'Develop, deploy, observe, and operate each service consistently.',
  },
];

type FeatureItem = {
  title: string;
  description: string;
};

function Feature({title, description}: FeatureItem) {
  return (
    <div className="col col--4">
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className="padding-vert--lg">
      <div className="container">
        <div className="row">
          {FeatureList.map((feature) => (
            <Feature key={feature.title} {...feature} />
          ))}
        </div>
      </div>
    </section>
  );
}
